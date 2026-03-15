package admin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodgate/internal/domain"
	"github.com/woodleighschool/woodgate/internal/store/db"
	pgutil "github.com/woodleighschool/woodgate/internal/store/postgres/shared"
)

func (store *Store) ListAPIKeys(ctx context.Context, options domain.APIKeyListOptions) ([]domain.APIKey, int32, error) {
	orderBy, err := pgutil.OrderBy(options.Sort, options.Order, map[string]string{
		"id":           "ak.id",
		"name":         "ak.name",
		"key_prefix":   "ak.key_prefix",
		"last_used_at": "ak.last_used_at",
		"expires_at":   "ak.expires_at",
		"created_at":   "ak.created_at",
	}, []string{"ak.created_at DESC", "ak.id ASC"})
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
SELECT
  ak.id,
  ak.name,
  ak.key_prefix,
  ak.key_hash,
  ak.last_used_at,
  ak.expires_at,
  ak.created_at,
  COUNT(*) OVER()::INT4 AS total
FROM api_keys AS ak
WHERE ($1 = '' OR ak.name ILIKE $1 OR ak.key_prefix ILIKE $1)
ORDER BY %s
LIMIT NULLIF($2::INT, 0)
OFFSET $3
`, orderBy)

	rows, err := store.store.Pool().
		Query(ctx, query, pgutil.SearchPattern(options.Search), options.Limit, options.Offset)
	if err != nil {
		return nil, 0, err
	}

	items, total, err := pgutil.CollectRows(rows, func(rows pgx.Rows) (domain.APIKey, int32, error) {
		var (
			row   db.ApiKey
			total int32
		)

		if scanErr := rows.Scan(
			&row.ID,
			&row.Name,
			&row.KeyPrefix,
			&row.KeyHash,
			&row.LastUsedAt,
			&row.ExpiresAt,
			&row.CreatedAt,
			&total,
		); scanErr != nil {
			return domain.APIKey{}, 0, scanErr
		}

		return mapAPIKey(row), total, nil
	})
	if err != nil {
		return nil, 0, err
	}

	enriched, enrichErr := store.attachAPIKeyAccess(ctx, items)
	if enrichErr != nil {
		return nil, 0, enrichErr
	}

	return enriched, total, nil
}

func (store *Store) CreateAPIKey(
	ctx context.Context,
	name string,
	prefix string,
	hash string,
	expiresAt *time.Time,
) (domain.APIKey, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return domain.APIKey{}, fmt.Errorf("create api key id: %w", err)
	}

	row, err := store.queries.CreateAPIKey(ctx, db.CreateAPIKeyParams{
		ID:        id,
		Name:      name,
		KeyPrefix: prefix,
		KeyHash:   hash,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return domain.APIKey{}, err
	}

	return mapAPIKey(row), nil
}

func (store *Store) GetAPIKey(ctx context.Context, id uuid.UUID) (domain.APIKey, error) {
	row, err := store.queries.GetAPIKey(ctx, id)
	if err != nil {
		return domain.APIKey{}, err
	}

	item := mapAPIKey(row)
	permissions, permissionsErr := store.listPermissionsBySubjects(
		ctx,
		domain.PermissionSubjectKindAPIKey,
		[]uuid.UUID{id},
	)
	if permissionsErr != nil {
		return domain.APIKey{}, permissionsErr
	}

	item.Access = permissions[id]
	admins, adminsErr := store.listAdminBySubjects(ctx, domain.PermissionSubjectKindAPIKey, []uuid.UUID{id})
	if adminsErr != nil {
		return domain.APIKey{}, adminsErr
	}

	item.Admin = admins[id]
	return item, nil
}

func (store *Store) UpdateAPIKeyAccess(
	ctx context.Context,
	id uuid.UUID,
	admin bool,
	access []domain.PermissionGrant,
) (domain.APIKey, error) {
	if _, err := store.queries.GetAPIKey(ctx, id); err != nil {
		return domain.APIKey{}, err
	}

	replaceErr := store.replacePrincipalAccess(
		ctx,
		domain.PermissionSubjectKindAPIKey,
		id,
		admin,
		access,
	)
	if replaceErr != nil {
		return domain.APIKey{}, replaceErr
	}

	return store.GetAPIKey(ctx, id)
}

func (store *Store) DeleteAPIKey(ctx context.Context, id uuid.UUID) error {
	_, err := store.queries.DeleteAPIKey(ctx, id)
	return err
}

func (store *Store) AuthenticateAPIKey(ctx context.Context, secret string) (uuid.UUID, error) {
	trimmed := strings.TrimSpace(secret)
	if trimmed == "" {
		return uuid.Nil, domain.ErrAPIKeyUnauthorized
	}

	hash := sha256.Sum256([]byte(trimmed))
	row, err := store.queries.GetAPIKeyByHash(ctx, hex.EncodeToString(hash[:]))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, domain.ErrAPIKeyUnauthorized
		}
		return uuid.Nil, err
	}

	if row.ExpiresAt != nil && !row.ExpiresAt.After(time.Now().UTC()) {
		return uuid.Nil, domain.ErrAPIKeyUnauthorized
	}

	updateErr := store.queries.UpdateAPIKeyLastUsedAt(ctx, row.ID)
	if updateErr != nil {
		return uuid.Nil, updateErr
	}

	return row.ID, nil
}

func mapAPIKey(row db.ApiKey) domain.APIKey {
	return domain.APIKey{
		ID:         row.ID,
		Name:       row.Name,
		KeyPrefix:  row.KeyPrefix,
		LastUsedAt: row.LastUsedAt,
		ExpiresAt:  row.ExpiresAt,
		Admin:      false,
		Access:     []domain.PermissionGrant{},
		CreatedAt:  row.CreatedAt,
	}
}

func (store *Store) attachAPIKeyAccess(ctx context.Context, items []domain.APIKey) ([]domain.APIKey, error) {
	subjectIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		subjectIDs = append(subjectIDs, item.ID)
	}

	permissionsBySubject, err := store.listPermissionsBySubjects(ctx, domain.PermissionSubjectKindAPIKey, subjectIDs)
	if err != nil {
		return nil, err
	}
	adminsBySubject, adminsErr := store.listAdminBySubjects(ctx, domain.PermissionSubjectKindAPIKey, subjectIDs)
	if adminsErr != nil {
		return nil, adminsErr
	}

	for index := range items {
		items[index].Admin = adminsBySubject[items[index].ID]
		items[index].Access = permissionsBySubject[items[index].ID]
	}

	return items, nil
}
