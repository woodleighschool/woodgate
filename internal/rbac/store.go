package rbac

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/goodies/auth/authz"

	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/postgres"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

const roleColumns = `id, key, name, description, builtin, created_at, updated_at`

func (store *Store) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := store.pool.Query(ctx, `SELECT `+roleColumns+` FROM authz_roles ORDER BY builtin DESC, lower(name), id`)
	if err != nil {
		return nil, err
	}
	roles, err := pgx.CollectRows(rows, pgx.RowToStructByName[Role])
	if err != nil {
		return nil, err
	}
	if err := store.attachPermissions(ctx, roles); err != nil {
		return nil, err
	}
	return roles, nil
}

func (store *Store) GetRole(ctx context.Context, id int64) (*Role, error) {
	role, err := postgres.GetOne[Role](ctx, store.pool, `SELECT `+roleColumns+` FROM authz_roles WHERE id = $1`, id)
	if err != nil {
		return nil, postgres.GetError(err)
	}
	roles := []Role{role}
	if err := store.attachPermissions(ctx, roles); err != nil {
		return nil, err
	}
	return &roles[0], nil
}

func (store *Store) CreateRole(ctx context.Context, mutation RoleMutation) (*Role, error) {
	mutation.normalize()
	if err := mutation.validate(true); err != nil {
		return nil, err
	}
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	role, err := postgres.GetOne[Role](ctx, tx, `
INSERT INTO authz_roles (key, name, description)
VALUES ($1, $2, $3)
RETURNING `+roleColumns, mutation.Key, mutation.Name, mutation.Description)
	if err != nil {
		return nil, postgres.MutationError(err)
	}
	if err := replacePermissions(ctx, tx, role.ID, mutation.Permissions); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	role.Permissions = mutation.Permissions
	return &role, nil
}

func (store *Store) UpdateRole(ctx context.Context, id int64, mutation RoleMutation) (*Role, error) {
	mutation.normalize()
	if err := mutation.validate(false); err != nil {
		return nil, err
	}
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	role, err := postgres.GetOne[Role](ctx, tx, `
UPDATE authz_roles
SET name = $2, description = $3, updated_at = now()
WHERE id = $1 AND NOT builtin
RETURNING `+roleColumns, id, mutation.Name, mutation.Description)
	if err != nil {
		if errors.Is(err, fault.ErrNotFound) {
			var builtin bool
			if lookupErr := tx.QueryRow(ctx, `SELECT builtin FROM authz_roles WHERE id = $1`, id).Scan(&builtin); lookupErr == nil && builtin {
				return nil, fmt.Errorf("%w: builtin roles cannot be changed", fault.ErrConflict)
			}
		}
		return nil, postgres.MutationError(err)
	}
	if err := replacePermissions(ctx, tx, role.ID, mutation.Permissions); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	role.Permissions = mutation.Permissions
	return &role, nil
}

func (store *Store) DeleteRole(ctx context.Context, id int64) error {
	var builtin bool
	if err := store.pool.QueryRow(ctx, `SELECT builtin FROM authz_roles WHERE id = $1`, id).Scan(&builtin); err != nil {
		return postgres.GetError(err)
	}
	if builtin {
		return fmt.Errorf("%w: builtin roles cannot be deleted", fault.ErrConflict)
	}
	tag, err := store.pool.Exec(ctx, `DELETE FROM authz_roles WHERE id = $1`, id)
	if err != nil {
		return postgres.MutationError(err)
	}
	if tag.RowsAffected() == 0 {
		return fault.ErrNotFound
	}
	return nil
}

func replacePermissions(ctx context.Context, tx pgx.Tx, roleID int64, permissions map[authz.Resource]authz.Access) error {
	if _, err := tx.Exec(ctx, `DELETE FROM authz_role_permissions WHERE role_id = $1`, roleID); err != nil {
		return err
	}
	for resource, access := range permissions {
		if access == authz.None {
			continue
		}
		level, err := accessLevel(access)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO authz_role_permissions (role_id, resource, access)
VALUES ($1, $2, $3)`, roleID, resource, level); err != nil {
			return postgres.MutationError(err)
		}
	}
	return nil
}

func (store *Store) attachPermissions(ctx context.Context, roles []Role) error {
	if len(roles) == 0 {
		return nil
	}
	ids := make([]int64, len(roles))
	byID := make(map[int64]*Role, len(roles))
	for i := range roles {
		ids[i] = roles[i].ID
		roles[i].Permissions = map[authz.Resource]authz.Access{}
		byID[roles[i].ID] = &roles[i]
	}
	rows, err := store.pool.Query(ctx, `
SELECT role_id, resource, access
FROM authz_role_permissions
WHERE role_id = ANY($1::bigint[])
ORDER BY role_id, resource`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var roleID int64
		var resource authz.Resource
		var level int16
		if err := rows.Scan(&roleID, &resource, &level); err != nil {
			return err
		}
		access, err := accessFromLevel(level)
		if err != nil {
			return err
		}
		byID[roleID].Permissions[resource] = access
	}
	return rows.Err()
}

// Grants returns direct and group-inherited role facts without evaluating them.
func (store *Store) Grants(ctx context.Context, userID int64) ([]authz.Grant, error) {
	rows, err := store.pool.Query(ctx, `
WITH assigned_roles AS (
    SELECT role_id FROM authz_user_roles WHERE user_id = $1
    UNION ALL
    SELECT group_role.role_id
    FROM authz_group_roles AS group_role
    JOIN directory_group_memberships AS membership ON membership.group_id = group_role.group_id
    WHERE membership.user_id = $1
)
SELECT permissions.resource, permissions.access
FROM assigned_roles AS roles
JOIN authz_role_permissions AS permissions ON permissions.role_id = roles.role_id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	grants := []authz.Grant{}
	for rows.Next() {
		var grant authz.Grant
		var level int16
		if err := rows.Scan(&grant.Resource, &level); err != nil {
			return nil, err
		}
		grant.Access, err = accessFromLevel(level)
		if err != nil {
			return nil, err
		}
		grants = append(grants, grant)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return grants, nil
}

func accessFromLevel(level int16) (authz.Access, error) {
	switch level {
	case 1:
		return authz.View, nil
	case 2:
		return authz.Edit, nil
	default:
		return "", fmt.Errorf("%w: database level %d", authz.ErrInvalidAccess, level)
	}
}

func accessLevel(access authz.Access) (int16, error) {
	switch access {
	case authz.None:
		return 0, nil
	case authz.View:
		return 1, nil
	case authz.Edit:
		return 2, nil
	default:
		return 0, fmt.Errorf("%w: %q", authz.ErrInvalidAccess, access)
	}
}
