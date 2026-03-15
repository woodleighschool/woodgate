package admin

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodgate/internal/domain"
	"github.com/woodleighschool/woodgate/internal/store/db"
	pgutil "github.com/woodleighschool/woodgate/internal/store/postgres/shared"
)

func (store *Store) ListLocations(
	ctx context.Context,
	options domain.LocationListOptions,
) ([]domain.Location, int32, error) {
	orderBy, err := pgutil.OrderBy(options.Sort, options.Order, map[string]string{
		"id":         "l.id",
		"name":       "l.name",
		"enabled":    "l.enabled",
		"notes":      "l.notes",
		"photo":      "l.photo",
		"created_at": "l.created_at",
		"updated_at": "l.updated_at",
	}, []string{"l.name ASC", "l.id ASC"})
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
SELECT
  l.id,
  l.name,
  l.description,
  l.enabled,
  l.notes,
  l.photo,
  l.background_asset_id,
  l.logo_asset_id,
  l.group_ids,
  l.created_at,
  l.updated_at,
  COUNT(*) OVER()::INT4 AS total
FROM locations AS l
WHERE (
  $1::BOOLEAN IS NULL
  OR l.enabled = $1::BOOLEAN
)
AND (
  $2 = ''
  OR l.name ILIKE $2
  OR l.description ILIKE $2
)
ORDER BY %s
LIMIT NULLIF($3::INT, 0)
OFFSET $4
`, orderBy)

	rows, err := store.store.Pool().Query(
		ctx,
		query,
		boolPointerValue(options.Enabled),
		pgutil.SearchPattern(options.Search),
		options.Limit,
		options.Offset,
	)
	if err != nil {
		return nil, 0, err
	}

	return pgutil.CollectRows(rows, func(rows pgx.Rows) (domain.Location, int32, error) {
		var (
			row   db.Location
			total int32
		)

		if scanErr := rows.Scan(
			&row.ID,
			&row.Name,
			&row.Description,
			&row.Enabled,
			&row.Notes,
			&row.Photo,
			&row.BackgroundAssetID,
			&row.LogoAssetID,
			&row.GroupIds,
			&row.CreatedAt,
			&row.UpdatedAt,
			&total,
		); scanErr != nil {
			return domain.Location{}, 0, scanErr
		}

		return mapLocation(row), total, nil
	})
}

func (store *Store) CreateLocation(
	ctx context.Context,
	name string,
	description string,
	enabled bool,
	notes bool,
	photo bool,
	backgroundAssetID *uuid.UUID,
	logoAssetID *uuid.UUID,
	groupIDs []uuid.UUID,
) (domain.Location, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return domain.Location{}, fmt.Errorf("create location id: %w", err)
	}

	row, err := store.queries.CreateLocation(ctx, db.CreateLocationParams{
		ID:                id,
		Name:              name,
		Description:       description,
		Enabled:           enabled,
		Notes:             notes,
		Photo:             photo,
		BackgroundAssetID: backgroundAssetID,
		LogoAssetID:       logoAssetID,
		GroupIds:          groupIDs,
	})
	if err != nil {
		return domain.Location{}, err
	}

	return mapLocation(row), nil
}

func (store *Store) GetLocation(ctx context.Context, id uuid.UUID) (domain.Location, error) {
	row, err := store.queries.GetLocation(ctx, id)
	if err != nil {
		return domain.Location{}, err
	}

	return mapLocation(row), nil
}

func (store *Store) UpdateLocation(
	ctx context.Context,
	id uuid.UUID,
	name string,
	description string,
	enabled bool,
	notes bool,
	photo bool,
	backgroundAssetID *uuid.UUID,
	logoAssetID *uuid.UUID,
	groupIDs []uuid.UUID,
) (domain.Location, error) {
	row, err := store.queries.UpdateLocation(ctx, db.UpdateLocationParams{
		ID:                id,
		Name:              name,
		Description:       description,
		Enabled:           enabled,
		Notes:             notes,
		Photo:             photo,
		BackgroundAssetID: backgroundAssetID,
		LogoAssetID:       logoAssetID,
		GroupIds:          groupIDs,
	})
	if err != nil {
		return domain.Location{}, err
	}

	return mapLocation(row), nil
}

func (store *Store) DeleteLocation(ctx context.Context, id uuid.UUID) error {
	_, err := store.queries.DeleteLocation(ctx, id)
	return err
}

func mapLocation(row db.Location) domain.Location {
	return domain.Location{
		ID:                row.ID,
		Name:              row.Name,
		Description:       row.Description,
		Enabled:           row.Enabled,
		Notes:             row.Notes,
		Photo:             row.Photo,
		BackgroundAssetID: row.BackgroundAssetID,
		LogoAssetID:       row.LogoAssetID,
		GroupIDs:          row.GroupIds,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func boolPointerValue(value *bool) any {
	if value == nil {
		return nil
	}
	return *value
}
