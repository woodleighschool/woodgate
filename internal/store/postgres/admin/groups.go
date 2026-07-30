package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodgate/internal/domain"
	"github.com/woodleighschool/woodgate/internal/store/db"
	pgutil "github.com/woodleighschool/woodgate/internal/store/postgres/shared"
)

func (store *Store) ListGroups(ctx context.Context, options domain.GroupListOptions) ([]domain.Group, int32, error) {
	orderBy, err := pgutil.OrderBy(options.Sort, options.Order, map[string]string{
		"id":           "g.id",
		"name":         "g.name",
		"description":  "g.description",
		"member_count": "member_count",
		"created_at":   "g.created_at",
		"updated_at":   "g.updated_at",
	}, []string{"g.name ASC", "g.id ASC"})
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
SELECT
  g.id,
  g.name,
  g.description,
  COALESCE(member_counts.member_count, 0)::INT4 AS member_count,
  g.created_at,
  g.updated_at,
  COUNT(*) OVER()::INT4 AS total
FROM groups AS g
LEFT JOIN (
  SELECT group_id, COUNT(*)::INT4 AS member_count
  FROM group_memberships
  GROUP BY group_id
) AS member_counts ON member_counts.group_id = g.id
WHERE (
  $1 = ''
  OR g.name ILIKE $1
  OR g.description ILIKE $1
)
ORDER BY %s
LIMIT NULLIF($2::INT, 0)
OFFSET $3
`, orderBy)

	rows, err := store.store.Pool().
		Query(ctx, query, pgutil.SearchPattern(options.Search), options.Limit, options.Offset)
	if err != nil {
		return nil, 0, err
	}

	return pgutil.CollectRows(rows, func(rows pgx.Rows) (domain.Group, int32, error) {
		var (
			row         db.Group
			memberCount int32
			total       int32
		)

		if scanErr := rows.Scan(
			&row.ID,
			&row.Name,
			&row.Description,
			&memberCount,
			&row.CreatedAt,
			&row.UpdatedAt,
			&total,
		); scanErr != nil {
			return domain.Group{}, 0, scanErr
		}

		return domain.Group{
			ID:          row.ID,
			Name:        row.Name,
			Description: row.Description,
			MemberCount: memberCount,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		}, total, nil
	})
}

func (store *Store) GetGroup(ctx context.Context, id uuid.UUID) (domain.Group, error) {
	return pgutil.GetGroup(ctx, store.queries, id)
}

func (store *Store) ListGroupMemberships(
	ctx context.Context,
	options domain.GroupMembershipListOptions,
) ([]domain.GroupMembership, int32, error) {
	orderBy, err := pgutil.OrderBy(options.Sort, options.Order, map[string]string{
		"id":         "gm.id",
		"group_id":   "gm.group_id",
		"user_id":    "gm.user_id",
		"created_at": "gm.created_at",
		"updated_at": "gm.updated_at",
	}, []string{"gm.group_id ASC", "gm.user_id ASC", "gm.id ASC"})
	if err != nil {
		return nil, 0, err
	}

	rows, err := store.store.Pool().Query(
		ctx,
		listGroupMembershipsQuery(orderBy),
		uuidOrNil(options.GroupID),
		uuidOrNil(options.UserID),
		pgutil.SearchPattern(options.Search),
		options.Limit,
		options.Offset,
	)
	if err != nil {
		return nil, 0, err
	}

	return pgutil.CollectRows(rows, scanGroupMembershipRow)
}

func (store *Store) GetGroupMembership(ctx context.Context, id uuid.UUID) (domain.GroupMembership, error) {
	row, err := store.queries.GetGroupMembership(ctx, id)
	if err != nil {
		return domain.GroupMembership{}, err
	}

	return domain.GroupMembership{
		ID:        row.ID,
		GroupID:   row.GroupID,
		UserID:    row.UserID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func listGroupMembershipsQuery(orderBy string) string {
	return fmt.Sprintf(`
SELECT
  gm.id,
  gm.group_id,
  gm.user_id,
  gm.created_at,
  gm.updated_at,
  COUNT(*) OVER()::INT4 AS total
FROM group_memberships AS gm
WHERE (
  $1::UUID IS NULL
  OR gm.group_id = $1::UUID
)
AND (
  $2::UUID IS NULL
  OR gm.user_id = $2::UUID
)
AND (
  $3 = ''
  OR gm.group_id::TEXT ILIKE $3
  OR gm.user_id::TEXT ILIKE $3
)
ORDER BY %s
LIMIT NULLIF($4::INT, 0)
OFFSET $5
`, orderBy)
}

func scanGroupMembershipRow(rows pgx.Rows) (domain.GroupMembership, int32, error) {
	var (
		id        uuid.UUID
		groupID   uuid.UUID
		userID    uuid.UUID
		createdAt time.Time
		updatedAt time.Time
		total     int32
	)

	if scanErr := rows.Scan(
		&id,
		&groupID,
		&userID,
		&createdAt,
		&updatedAt,
		&total,
	); scanErr != nil {
		return domain.GroupMembership{}, 0, scanErr
	}

	return domain.GroupMembership{
		ID:        id,
		GroupID:   groupID,
		UserID:    userID,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, total, nil
}
