package admin

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodgate/internal/domain"
	"github.com/woodleighschool/woodgate/internal/store/db"
	pgutil "github.com/woodleighschool/woodgate/internal/store/postgres/shared"
)

func (store *Store) ListCheckins(
	ctx context.Context,
	options domain.CheckinListOptions,
	allowedLocationIDs []uuid.UUID,
) ([]domain.Checkin, int32, error) {
	if allowedLocationIDs != nil && len(allowedLocationIDs) == 0 {
		return []domain.Checkin{}, 0, nil
	}

	orderBy, err := pgutil.OrderBy(options.Sort, options.Order, map[string]string{
		"id":          "c.id",
		"user_id":     "c.user_id",
		"location_id": "c.location_id",
		"direction":   "c.direction",
		"created_at":  "c.created_at",
	}, []string{"c.created_at DESC", "c.id DESC"})
	if err != nil {
		return nil, 0, err
	}

	rows, err := store.store.Pool().Query(
		ctx,
		listCheckinsQuery(orderBy),
		uuidSliceOrNil(allowedLocationIDs),
		uuidOrNil(options.LocationID),
		uuidOrNil(options.UserID),
		stringValue(options.Direction),
		timePointerValue(options.CreatedFrom),
		timePointerValue(options.CreatedTo),
		pgutil.SearchPattern(options.Search),
		options.Limit,
		options.Offset,
	)
	if err != nil {
		return nil, 0, err
	}

	return pgutil.CollectRows(rows, scanCheckinRow)
}

func (store *Store) CreateCheckin(
	ctx context.Context,
	userID uuid.UUID,
	locationID uuid.UUID,
	direction domain.CheckinDirection,
	notes string,
	assetID *uuid.UUID,
	createdByKind domain.PermissionSubjectKind,
	createdByID uuid.UUID,
) (domain.Checkin, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return domain.Checkin{}, fmt.Errorf("create checkin id: %w", err)
	}

	_, createErr := store.queries.CreateCheckin(ctx, db.CreateCheckinParams{
		ID:            id,
		UserID:        userID,
		LocationID:    locationID,
		Direction:     string(direction),
		Notes:         notes,
		AssetID:       assetID,
		CreatedByKind: string(createdByKind),
		CreatedByID:   createdByID,
	})
	if createErr != nil {
		return domain.Checkin{}, createErr
	}

	return store.GetCheckin(ctx, id, []uuid.UUID{locationID})
}

func (store *Store) GetCheckin(
	ctx context.Context,
	id uuid.UUID,
	allowedLocationIDs []uuid.UUID,
) (domain.Checkin, error) {
	row, err := store.queries.GetCheckin(ctx, id)
	if err != nil {
		return domain.Checkin{}, err
	}

	if allowedLocationIDs != nil && !containsUUID(allowedLocationIDs, row.LocationID) {
		return domain.Checkin{}, pgx.ErrNoRows
	}

	direction, err := pgutil.ToCheckinDirection(row.Direction)
	if err != nil {
		return domain.Checkin{}, err
	}
	subjectKind, err := pgutil.ToPermissionSubjectKind(row.CreatedByKind)
	if err != nil {
		return domain.Checkin{}, err
	}

	return domain.Checkin{
		ID:            row.ID,
		UserID:        row.UserID,
		LocationID:    row.LocationID,
		Direction:     direction,
		Notes:         row.Notes,
		AssetID:       row.AssetID,
		CreatedByKind: subjectKind,
		CreatedByID:   row.CreatedByID,
		CreatedAt:     row.CreatedAt,
	}, nil
}

func listCheckinsQuery(orderBy string) string {
	return fmt.Sprintf(`
SELECT
  c.id,
  c.user_id,
  c.location_id,
  c.direction,
  c.notes,
  c.asset_id,
  c.created_by_kind,
  c.created_by_id,
  c.created_at,
  COUNT(*) OVER()::INT4 AS total
FROM checkins AS c
WHERE ($1::UUID[] IS NULL OR c.location_id = ANY($1::UUID[]))
  AND ($2::UUID IS NULL OR c.location_id = $2::UUID)
  AND ($3::UUID IS NULL OR c.user_id = $3::UUID)
  AND ($4 = '' OR c.direction = $4)
  AND ($5::TIMESTAMPTZ IS NULL OR c.created_at >= $5::TIMESTAMPTZ)
  AND ($6::TIMESTAMPTZ IS NULL OR c.created_at <= $6::TIMESTAMPTZ)
  AND (
    $7 = ''
    OR c.user_id::TEXT ILIKE $7
    OR c.location_id::TEXT ILIKE $7
    OR c.notes ILIKE $7
  )
ORDER BY %s
LIMIT NULLIF($8::INT, 0)
OFFSET $9
`, orderBy)
}

func scanCheckinRow(rows pgx.Rows) (domain.Checkin, int32, error) {
	var (
		id            uuid.UUID
		userID        uuid.UUID
		locationID    uuid.UUID
		directionRaw  string
		notes         string
		assetID       *uuid.UUID
		createdByKind string
		createdByID   uuid.UUID
		createdAt     time.Time
		total         int32
	)

	if scanErr := rows.Scan(
		&id,
		&userID,
		&locationID,
		&directionRaw,
		&notes,
		&assetID,
		&createdByKind,
		&createdByID,
		&createdAt,
		&total,
	); scanErr != nil {
		return domain.Checkin{}, 0, scanErr
	}

	direction, directionErr := pgutil.ToCheckinDirection(directionRaw)
	if directionErr != nil {
		return domain.Checkin{}, 0, directionErr
	}
	subjectKind, subjectKindErr := pgutil.ToPermissionSubjectKind(createdByKind)
	if subjectKindErr != nil {
		return domain.Checkin{}, 0, subjectKindErr
	}

	return domain.Checkin{
		ID:            id,
		UserID:        userID,
		LocationID:    locationID,
		Direction:     direction,
		Notes:         notes,
		AssetID:       assetID,
		CreatedByKind: subjectKind,
		CreatedByID:   createdByID,
		CreatedAt:     createdAt,
	}, total, nil
}

func timePointerValue(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func containsUUID(values []uuid.UUID, target uuid.UUID) bool {
	return slices.Contains(values, target)
}

func uuidSliceOrNil(values []uuid.UUID) any {
	if values == nil {
		return nil
	}
	return values
}

func stringValue[T ~string](value *T) string {
	if value == nil {
		return ""
	}
	return string(*value)
}
