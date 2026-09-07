package directory

import (
	"context"

	"github.com/woodleighschool/woodgate/internal/listing"
	"github.com/woodleighschool/woodgate/internal/postgres"
)

func (s *Store) ListGroups(ctx context.Context, params GroupListParams) ([]Group, int, error) {
	params.ListParams = listing.Normalize(params.ListParams)
	params.Values = listing.NormalizeValues(params.Values)
	where, args := groupWhere(params)
	groups, count, err := postgres.ListWithCount[Group](ctx, s.pool, groupListQuery(params, where, args))
	if err != nil {
		return nil, 0, err
	}
	if err := attachGroupRoles(ctx, s.pool, groups); err != nil {
		return nil, 0, err
	}
	return groups, count, nil
}

func (s *Store) GetGroupByID(ctx context.Context, id int64) (*Group, error) {
	group, err := postgres.GetOne[Group](ctx, s.pool, groupSelectSQL()+`WHERE g.id = $1
GROUP BY g.id`, id)
	if err != nil {
		return nil, postgres.GetError(err)
	}
	groups := []Group{group}
	if err := attachGroupRoles(ctx, s.pool, groups); err != nil {
		return nil, err
	}
	group = groups[0]
	return &group, nil
}

// UpdateGroup replaces the application-owned fields on a source-owned group.
func (s *Store) UpdateGroup(ctx context.Context, id int64, mutation GroupMutation) (*Group, error) {
	mutation.normalize()
	if err := mutation.validate(); err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var groupID int64
	if err := tx.QueryRow(ctx, `UPDATE directory_groups
SET updated_at = now()
WHERE id = $1
RETURNING id`, id).Scan(&groupID); err != nil {
		return nil, postgres.MutationError(err)
	}
	if err := replaceGroupRoles(ctx, tx, groupID, mutation.RoleIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetGroupByID(ctx, groupID)
}

func groupSelectSQL() string {
	return `SELECT
		g.id,
		g.source::text AS source,
		g.external_id,
		g.display_name,
		COALESCE(g.mail_nickname, '') AS mail_nickname,
		count(u.id)::integer AS member_count,
		g.created_at,
		g.updated_at
	FROM directory_groups g
	LEFT JOIN directory_group_memberships gm ON gm.group_id = g.id
	LEFT JOIN users u ON u.id = gm.user_id AND u.deleted_at IS NULL
	`
}

func groupWhere(params GroupListParams) (string, []any) {
	var where postgres.WhereBuilder
	if params.ListParams.Q != "" {
		search := where.Arg("%" + params.ListParams.Q + "%")
		where.Add(`(
			g.display_name ILIKE ` + search + `
			OR g.mail_nickname ILIKE ` + search + `
			OR g.external_id ILIKE ` + search + `
		)`)
	}
	if len(params.Values) > 0 {
		values := where.Arg(listing.NormalizeValues(params.Values))
		where.Add("g.id::text = ANY(" + values + "::text[])")
	}
	return where.Build()
}

func groupListQuery(params GroupListParams, where string, args []any) postgres.ListQuery {
	return postgres.ListQuery{
		SelectSQL:  groupSelectSQL(),
		WhereSQL:   where,
		GroupBySQL: "GROUP BY g.id",
		Args:       args,
		OrderKeys: map[string]postgres.OrderExpr{
			"display_name":  {SQL: "lower(g.display_name)"},
			"mail_nickname": {SQL: "lower(g.mail_nickname)", NullOrder: postgres.NullsLast},
			"member_count":  {SQL: "member_count"},
			"source":        {SQL: "g.source"},
		},
		DefaultOrder: []postgres.OrderExpr{{SQL: "lower(g.display_name)"}, {SQL: "g.id"}},
		Params:       params.ListParams,
	}
}
