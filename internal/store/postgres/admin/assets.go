package admin

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/woodleighschool/woodgate/internal/domain"
	"github.com/woodleighschool/woodgate/internal/store/db"
	pgutil "github.com/woodleighschool/woodgate/internal/store/postgres/shared"
)

func (store *Store) ListAssets(ctx context.Context, options domain.AssetListOptions) ([]domain.Asset, int32, error) {
	orderBy, err := pgutil.OrderBy(options.Sort, options.Order, map[string]string{
		"id":         "a.id",
		"name":       "a.name",
		"type":       "a.type",
		"created_at": "a.created_at",
		"updated_at": "a.updated_at",
	}, []string{"a.created_at DESC", "a.id ASC"})
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
SELECT
  a.id,
  a.name,
  a.type,
  a.content_type,
  a.file_extension,
  a.created_at,
  a.updated_at,
  COUNT(*) OVER()::INT4 AS total
FROM assets AS a
WHERE (
  COALESCE(array_length($1::TEXT[], 1), 0) = 0
  OR a.type = ANY($1::TEXT[])
)
AND (
  $2 = ''
  OR COALESCE(a.name, '') ILIKE $2
  OR a.type ILIKE $2
)
ORDER BY %s
LIMIT NULLIF($3::INT, 0)
OFFSET $4
`, orderBy)

	rows, err := store.store.Pool().
		Query(ctx, query, assetTypesOrNil(options.Types), pgutil.SearchPattern(options.Search), options.Limit, options.Offset)
	if err != nil {
		return nil, 0, err
	}

	return pgutil.CollectRows(rows, func(rows pgx.Rows) (domain.Asset, int32, error) {
		var (
			row   db.Asset
			total int32
		)

		if scanErr := rows.Scan(
			&row.ID,
			&row.Name,
			&row.Type,
			&row.ContentType,
			&row.FileExtension,
			&row.CreatedAt,
			&row.UpdatedAt,
			&total,
		); scanErr != nil {
			return domain.Asset{}, 0, scanErr
		}

		assetType, assetTypeErr := pgutil.ToAssetType(row.Type)
		if assetTypeErr != nil {
			return domain.Asset{}, 0, assetTypeErr
		}

		return domain.Asset{
			ID:            row.ID,
			Name:          textPointer(row.Name),
			Type:          assetType,
			ContentType:   row.ContentType,
			FileExtension: row.FileExtension,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		}, total, nil
	})
}

func (store *Store) CreateAsset(
	ctx context.Context,
	id uuid.UUID,
	name *string,
	assetType domain.AssetType,
	contentType string,
	fileExtension string,
) (domain.Asset, error) {
	row, err := store.queries.CreateAsset(ctx, db.CreateAssetParams{
		ID:            id,
		Name:          pgText(name),
		Type:          string(assetType),
		ContentType:   contentType,
		FileExtension: fileExtension,
	})
	if err != nil {
		return domain.Asset{}, err
	}

	return mapAsset(row), nil
}

func (store *Store) GetAsset(ctx context.Context, id uuid.UUID) (domain.Asset, error) {
	row, err := store.queries.GetAsset(ctx, id)
	if err != nil {
		return domain.Asset{}, err
	}

	return mapAsset(row), nil
}

func (store *Store) UpdateAsset(
	ctx context.Context,
	id uuid.UUID,
	name *string,
	contentType string,
	fileExtension string,
) (domain.Asset, error) {
	row, err := store.queries.UpdateAsset(ctx, db.UpdateAssetParams{
		ID:            id,
		Name:          pgText(name),
		ContentType:   contentType,
		FileExtension: fileExtension,
	})
	if err != nil {
		return domain.Asset{}, err
	}

	return mapAsset(row), nil
}

func (store *Store) DeleteAsset(ctx context.Context, id uuid.UUID) error {
	_, err := store.queries.DeleteAsset(ctx, id)
	return err
}

func (store *Store) ListCheckinLocationIDsByAsset(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	return store.queries.ListCheckinLocationIDsByAsset(ctx, &id)
}

func mapAsset(row db.Asset) domain.Asset {
	assetType, _ := pgutil.ToAssetType(row.Type)

	return domain.Asset{
		ID:            row.ID,
		Name:          textPointer(row.Name),
		Type:          assetType,
		ContentType:   row.ContentType,
		FileExtension: row.FileExtension,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

func pgText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{String: *value, Valid: true}
}

func textPointer(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func assetTypePointer(value pgtype.Text) *domain.AssetType {
	if !value.Valid {
		return nil
	}
	assetType, err := domain.ParseAssetType(value.String)
	if err != nil {
		return nil
	}
	return &assetType
}

func assetTypesOrNil(values []domain.AssetType) any {
	if len(values) == 0 {
		return nil
	}

	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, string(value))
	}
	return items
}
