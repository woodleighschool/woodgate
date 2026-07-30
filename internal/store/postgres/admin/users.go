package admin

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/woodleighschool/woodgate/internal/domain"
	"github.com/woodleighschool/woodgate/internal/store/db"
	pgutil "github.com/woodleighschool/woodgate/internal/store/postgres/shared"
)

func (store *Store) ListUsers(ctx context.Context, options domain.UserListOptions) ([]domain.User, int32, error) {
	orderBy, err := pgutil.OrderBy(options.Sort, options.Order, map[string]string{
		"id":           "u.id",
		"display_name": "u.display_name",
		"department":   "u.department",
		"upn":          "u.upn",
		"source":       "u.source",
		"created_at":   "u.created_at",
		"updated_at":   "u.updated_at",
	}, []string{"u.display_name ASC", "u.id ASC"})
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
SELECT
  u.id,
  u.upn,
  u.display_name,
  u.department,
  u.source,
  u.created_at,
  u.updated_at,
  COUNT(*) OVER()::INT4 AS total
FROM users AS u
WHERE (
  $1::UUID IS NULL
  OR EXISTS (
    SELECT 1
    FROM locations AS l
    JOIN group_memberships AS gm ON gm.group_id = ANY(l.group_ids)
    WHERE l.id = $1::UUID
      AND gm.user_id = u.id
  )
)
AND (
  $2 = ''
  OR u.display_name ILIKE $2
  OR u.upn ILIKE $2
  OR u.department ILIKE $2
  OR u.source ILIKE $2
)
ORDER BY %s
LIMIT NULLIF($3::INT, 0)
OFFSET $4
`, orderBy)

	rows, err := store.store.Pool().
		Query(ctx, query, uuidOrNil(options.LocationID), pgutil.SearchPattern(options.Search), options.Limit, options.Offset)
	if err != nil {
		return nil, 0, err
	}

	items, total, err := pgutil.CollectRows(rows, func(rows pgx.Rows) (domain.User, int32, error) {
		var (
			row   db.User
			total int32
		)

		if scanErr := rows.Scan(
			&row.ID,
			&row.Upn,
			&row.DisplayName,
			&row.Department,
			&row.Source,
			&row.CreatedAt,
			&row.UpdatedAt,
			&total,
		); scanErr != nil {
			return domain.User{}, 0, scanErr
		}

		mapped, mapErr := pgutil.MapUserRow(row)
		if mapErr != nil {
			return domain.User{}, 0, mapErr
		}

		return mapped, total, nil
	})
	if err != nil {
		return nil, 0, err
	}

	enriched, enrichErr := store.attachUserAccess(ctx, items)
	if enrichErr != nil {
		return nil, 0, enrichErr
	}

	return enriched, total, nil
}

func (store *Store) GetUser(ctx context.Context, id uuid.UUID) (domain.User, error) {
	user, err := pgutil.GetUser(ctx, store.queries, id)
	if err != nil {
		return domain.User{}, err
	}

	permissions, permissionsErr := store.listPermissionsBySubjects(
		ctx,
		domain.PermissionSubjectKindUser,
		[]uuid.UUID{id},
	)
	if permissionsErr != nil {
		return domain.User{}, permissionsErr
	}

	user.Access = permissions[id]
	admins, adminsErr := store.listAdminBySubjects(ctx, domain.PermissionSubjectKindUser, []uuid.UUID{id})
	if adminsErr != nil {
		return domain.User{}, adminsErr
	}

	user.Admin = admins[id]
	return user, nil
}

func (store *Store) UpdateUserAccess(
	ctx context.Context,
	id uuid.UUID,
	admin bool,
	access []domain.PermissionGrant,
) (domain.User, error) {
	if _, err := store.queries.GetUser(ctx, id); err != nil {
		return domain.User{}, err
	}

	replaceErr := store.replacePrincipalAccess(
		ctx,
		domain.PermissionSubjectKindUser,
		id,
		admin,
		access,
	)
	if replaceErr != nil {
		return domain.User{}, replaceErr
	}

	return store.GetUser(ctx, id)
}

func (store *Store) GetUserByUPN(ctx context.Context, upn string) (domain.User, error) {
	row, err := store.queries.GetUserByUPN(ctx, strings.TrimSpace(upn))
	if err != nil {
		return domain.User{}, err
	}

	return pgutil.MapUserRow(row)
}

func mapPermissionGrant(row db.Permission) (domain.PermissionGrant, error) {
	resource, err := pgutil.ToPermissionResource(row.Resource)
	if err != nil {
		return domain.PermissionGrant{}, err
	}
	action, err := pgutil.ToPermissionAction(row.Action)
	if err != nil {
		return domain.PermissionGrant{}, err
	}

	return domain.PermissionGrant{
		Resource:   resource,
		Action:     action,
		LocationID: row.LocationID,
		AssetType:  assetTypePointer(row.AssetType),
	}, nil
}

func (store *Store) listPermissionsBySubjects(
	ctx context.Context,
	subjectKind domain.PermissionSubjectKind,
	subjectIDs []uuid.UUID,
) (map[uuid.UUID][]domain.PermissionGrant, error) {
	permissionsBySubject := make(map[uuid.UUID][]domain.PermissionGrant, len(subjectIDs))
	for _, subjectID := range subjectIDs {
		permissionsBySubject[subjectID] = []domain.PermissionGrant{}
	}
	if len(subjectIDs) == 0 {
		return permissionsBySubject, nil
	}

	rows, err := store.queries.ListPermissionsBySubjects(ctx, db.ListPermissionsBySubjectsParams{
		SubjectKind: string(subjectKind),
		SubjectIds:  subjectIDs,
	})
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		grant, grantErr := mapPermissionGrant(row)
		if grantErr != nil {
			return nil, grantErr
		}
		permissionsBySubject[row.SubjectID] = append(permissionsBySubject[row.SubjectID], grant)
	}

	return permissionsBySubject, nil
}

func (store *Store) listAdminBySubjects(
	ctx context.Context,
	subjectKind domain.PermissionSubjectKind,
	subjectIDs []uuid.UUID,
) (map[uuid.UUID]bool, error) {
	adminBySubject := make(map[uuid.UUID]bool, len(subjectIDs))
	for _, subjectID := range subjectIDs {
		adminBySubject[subjectID] = false
	}
	if len(subjectIDs) == 0 {
		return adminBySubject, nil
	}

	rows, err := store.queries.ListPrincipalRolesByPrincipals(ctx, db.ListPrincipalRolesByPrincipalsParams{
		PrincipalKind: string(subjectKind),
		PrincipalIds:  subjectIDs,
	})
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		if strings.EqualFold(strings.TrimSpace(row.Role), "admin") {
			adminBySubject[row.PrincipalID] = true
		}
	}

	return adminBySubject, nil
}

func (store *Store) ListPrincipalRoles(ctx context.Context) ([]domain.PrincipalRole, error) {
	rows, err := store.queries.ListPrincipalRoles(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]domain.PrincipalRole, 0, len(rows))
	for _, row := range rows {
		kind, kindErr := pgutil.ToPermissionSubjectKind(row.PrincipalKind)
		if kindErr != nil {
			return nil, kindErr
		}

		items = append(items, domain.PrincipalRole{
			PrincipalKind: kind,
			PrincipalID:   row.PrincipalID,
			Role:          row.Role,
		})
	}

	return items, nil
}

func (store *Store) ListPrincipalPermissionGrants(ctx context.Context) ([]domain.PrincipalPermissionGrant, error) {
	rows, err := store.queries.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]domain.PrincipalPermissionGrant, 0, len(rows))
	for _, row := range rows {
		kind, kindErr := pgutil.ToPermissionSubjectKind(row.SubjectKind)
		if kindErr != nil {
			return nil, kindErr
		}
		grant, grantErr := mapPermissionGrant(row)
		if grantErr != nil {
			return nil, grantErr
		}

		items = append(items, domain.PrincipalPermissionGrant{
			PrincipalKind: kind,
			PrincipalID:   row.SubjectID,
			Grant:         grant,
		})
	}

	return items, nil
}

func (store *Store) replacePrincipalAccess(
	ctx context.Context,
	subjectKind domain.PermissionSubjectKind,
	subjectID uuid.UUID,
	admin bool,
	access []domain.PermissionGrant,
) error {
	return store.store.RunInTx(ctx, func(queries *db.Queries) error {
		deleteRolesErr := queries.DeletePrincipalRolesByPrincipal(ctx, db.DeletePrincipalRolesByPrincipalParams{
			PrincipalKind: string(subjectKind),
			PrincipalID:   subjectID,
		})
		if deleteRolesErr != nil {
			return deleteRolesErr
		}

		deleteErr := queries.DeletePermissionsBySubject(ctx, db.DeletePermissionsBySubjectParams{
			SubjectKind: string(subjectKind),
			SubjectID:   subjectID,
		})
		if deleteErr != nil {
			return deleteErr
		}

		if admin {
			roleID, roleIDErr := uuid.NewV7()
			if roleIDErr != nil {
				return fmt.Errorf("create principal role id: %w", roleIDErr)
			}

			createRoleErr := queries.CreatePrincipalRole(ctx, db.CreatePrincipalRoleParams{
				ID:            roleID,
				PrincipalKind: string(subjectKind),
				PrincipalID:   subjectID,
				Role:          "admin",
			})
			if createRoleErr != nil {
				return createRoleErr
			}
		}

		for _, permission := range access {
			id, idErr := uuid.NewV7()
			if idErr != nil {
				return fmt.Errorf("create permission id: %w", idErr)
			}

			createErr := queries.CreatePermission(ctx, db.CreatePermissionParams{
				ID:          id,
				SubjectKind: string(subjectKind),
				SubjectID:   subjectID,
				Resource:    string(permission.Resource),
				Action:      string(permission.Action),
				LocationID:  permission.LocationID,
				AssetType:   textValue(permission.AssetType),
			})
			if createErr != nil {
				return createErr
			}
		}

		return nil
	})
}

func (store *Store) attachUserAccess(ctx context.Context, items []domain.User) ([]domain.User, error) {
	subjectIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		subjectIDs = append(subjectIDs, item.ID)
	}

	permissionsBySubject, err := store.listPermissionsBySubjects(ctx, domain.PermissionSubjectKindUser, subjectIDs)
	if err != nil {
		return nil, err
	}
	adminsBySubject, adminsErr := store.listAdminBySubjects(ctx, domain.PermissionSubjectKindUser, subjectIDs)
	if adminsErr != nil {
		return nil, adminsErr
	}

	for index := range items {
		items[index].Admin = adminsBySubject[items[index].ID]
		items[index].Access = permissionsBySubject[items[index].ID]
	}

	return items, nil
}

func uuidOrNil(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return *value
}

func textValue[T ~string](value *T) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: string(*value), Valid: true}
}
