package directory

import (
	"context"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/postgres"
)

type roleQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func normalizeRoleIDs(ids []int64) []int64 {
	ids = slices.Clone(ids)
	slices.Sort(ids)
	return slices.Compact(ids)
}

func validateRoleIDs(ids []int64) error {
	for _, id := range ids {
		if id <= 0 {
			return fmt.Errorf("%w: role_ids must be positive", fault.ErrInvalidInput)
		}
	}
	return nil
}

func attachUserRoles(ctx context.Context, queryer roleQueryer, users []User) error {
	if len(users) == 0 {
		return nil
	}
	ids := make([]int64, len(users))
	byID := make(map[int64]*User, len(users))
	for i := range users {
		ids[i] = users[i].ID
		users[i].Roles = []RoleSummary{}
		users[i].EffectiveRoles = []RoleSummary{}
		byID[users[i].ID] = &users[i]
	}
	rows, err := queryer.Query(ctx, `
WITH assigned AS (
    SELECT user_id, role_id, true AS direct
    FROM authz_user_roles
    WHERE user_id = ANY($1::bigint[])
    UNION ALL
    SELECT membership.user_id, group_role.role_id, false AS direct
    FROM directory_group_memberships AS membership
    JOIN authz_group_roles AS group_role ON group_role.group_id = membership.group_id
    WHERE membership.user_id = ANY($1::bigint[])
), effective AS (
    SELECT user_id, role_id, bool_or(direct) AS direct
    FROM assigned
    GROUP BY user_id, role_id
)
SELECT effective.user_id, role.id, role.key, role.name, effective.direct
FROM effective
JOIN authz_roles AS role ON role.id = effective.role_id
ORDER BY effective.user_id, role.builtin DESC, lower(role.name), role.id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var userID int64
		var role RoleSummary
		var direct bool
		if err := rows.Scan(&userID, &role.ID, &role.Key, &role.Name, &direct); err != nil {
			return err
		}
		user := byID[userID]
		user.EffectiveRoles = append(user.EffectiveRoles, role)
		if direct {
			user.Roles = append(user.Roles, role)
		}
	}
	return rows.Err()
}

func attachGroupRoles(ctx context.Context, queryer roleQueryer, groups []Group) error {
	if len(groups) == 0 {
		return nil
	}
	ids := make([]int64, len(groups))
	byID := make(map[int64]*Group, len(groups))
	for i := range groups {
		ids[i] = groups[i].ID
		groups[i].Roles = []RoleSummary{}
		byID[groups[i].ID] = &groups[i]
	}
	rows, err := queryer.Query(ctx, `
SELECT membership.group_id, role.id, role.key, role.name
FROM authz_group_roles AS membership
JOIN authz_roles AS role ON role.id = membership.role_id
WHERE membership.group_id = ANY($1::bigint[])
ORDER BY membership.group_id, role.builtin DESC, lower(role.name), role.id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var groupID int64
		var role RoleSummary
		if err := rows.Scan(&groupID, &role.ID, &role.Key, &role.Name); err != nil {
			return err
		}
		group := byID[groupID]
		group.Roles = append(group.Roles, role)
	}
	return rows.Err()
}

func replaceUserRoles(ctx context.Context, tx pgx.Tx, userID int64, roleIDs []int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM authz_user_roles WHERE user_id = $1`, userID); err != nil {
		return postgres.MutationError(err)
	}
	if len(roleIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
INSERT INTO authz_user_roles (user_id, role_id)
SELECT $1, unnest($2::bigint[])`, userID, roleIDs)
	return postgres.MutationError(err)
}

func replaceGroupRoles(ctx context.Context, tx pgx.Tx, groupID int64, roleIDs []int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM authz_group_roles WHERE group_id = $1`, groupID); err != nil {
		return postgres.MutationError(err)
	}
	if len(roleIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
INSERT INTO authz_group_roles (group_id, role_id)
SELECT $1, unnest($2::bigint[])`, groupID, roleIDs)
	return postgres.MutationError(err)
}
