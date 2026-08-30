package station

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/postgres"
)

const onlineWindow = 45 * time.Second

// Store persists stations and their last observed companion build.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns a PostgreSQL-backed station store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

type stationRow struct {
	ID              int64      `db:"id"`
	Name            string     `db:"name"`
	LocationID      int64      `db:"location_id"`
	Enabled         bool       `db:"enabled"`
	SecretPrefix    string     `db:"secret_prefix"`
	LastSeenAt      *time.Time `db:"last_seen_at"`
	AppBuild        string     `db:"app_build"`
	ProtocolVersion *int       `db:"protocol_version"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
}

const stationColumnsSQL = `id, name, location_id, enabled, secret_prefix,
    last_seen_at, app_build, protocol_version, created_at, updated_at`

const stationSelectSQL = `SELECT ` + stationColumnsSQL + `
FROM stations`

func (s *Store) List(ctx context.Context, params StationListParams) ([]Station, int, error) {
	where := postgres.WhereBuilder{}
	if params.ListParams.Q != "" {
		where.Addf("(name ILIKE '%%' || %s || '%%' OR secret_prefix ILIKE '%%' || %s || '%%')",
			params.ListParams.Q, params.ListParams.Q)
	}
	if params.LocationID > 0 {
		where.Addf("location_id = %s", params.LocationID)
	}
	if params.Enabled != nil {
		where.Addf("enabled = %s", *params.Enabled)
	}
	whereSQL, args := where.Build()
	rows, count, err := postgres.ListWithCount[stationRow](ctx, s.pool, postgres.ListQuery{
		SelectSQL: stationSelectSQL,
		WhereSQL:  whereSQL,
		Args:      args,
		OrderKeys: map[string]postgres.OrderExpr{
			"id":           {SQL: "id"},
			"name":         {SQL: "lower(name)"},
			"location_id":  {SQL: "location_id"},
			"enabled":      {SQL: "enabled"},
			"last_seen_at": {SQL: "last_seen_at", NullOrder: postgres.NullsLast},
			"app_build":    {SQL: "lower(app_build)"},
			"created_at":   {SQL: "created_at"},
			"updated_at":   {SQL: "updated_at"},
		},
		DefaultOrder: []postgres.OrderExpr{{SQL: "lower(name)"}, {SQL: "id"}},
		Params:       params.ListParams,
	})
	if err != nil {
		return nil, 0, err
	}
	stations := make([]Station, len(rows))
	for i, row := range rows {
		stations[i] = stationFromRow(row)
	}
	return stations, count, nil
}

func (s *Store) Get(ctx context.Context, id int64) (*Station, error) {
	row, err := postgres.GetOne[stationRow](ctx, s.pool, stationSelectSQL+"\nWHERE id = $1", id)
	if err != nil {
		return nil, postgres.GetError(err)
	}
	station := stationFromRow(row)
	return &station, nil
}

func (s *Store) create(
	ctx context.Context,
	mutation StationMutation,
	secretPrefix string,
	secretHash string,
) (*Station, error) {
	row, err := postgres.GetOne[stationRow](ctx, s.pool, `
INSERT INTO stations (name, location_id, enabled, secret_prefix, secret_hash)
VALUES ($1, $2, $3, $4, $5)
RETURNING `+stationColumnsSQL,
		mutation.Name, mutation.LocationID, mutation.Enabled, secretPrefix, secretHash)
	if err != nil {
		return nil, postgres.MutationError(err)
	}
	station := stationFromRow(row)
	return &station, nil
}

func (s *Store) update(ctx context.Context, id int64, mutation StationMutation) (*Station, error) {
	row, err := postgres.GetOne[stationRow](ctx, s.pool, `
UPDATE stations
SET name = $1,
    location_id = $2,
    enabled = $3,
    updated_at = now()
WHERE id = $4
RETURNING `+stationColumnsSQL,
		mutation.Name, mutation.LocationID, mutation.Enabled, id)
	if err != nil {
		return nil, postgres.MutationError(err)
	}
	station := stationFromRow(row)
	return &station, nil
}

func (s *Store) delete(ctx context.Context, id int64) error {
	var deletedID int64
	err := s.pool.QueryRow(ctx, `DELETE FROM stations WHERE id = $1 RETURNING id`, id).Scan(&deletedID)
	return postgres.DeleteConflict(err, "station is still referenced")
}

func (s *Store) rotateSecret(
	ctx context.Context,
	id int64,
	secretPrefix string,
	secretHash string,
) (*Station, error) {
	row, err := postgres.GetOne[stationRow](ctx, s.pool, `
UPDATE stations
SET secret_prefix = $1,
    secret_hash = $2,
    last_seen_at = NULL,
    app_build = '',
    protocol_version = NULL,
    updated_at = now()
WHERE id = $3
RETURNING `+stationColumnsSQL, secretPrefix, secretHash, id)
	if err != nil {
		return nil, postgres.MutationError(err)
	}
	station := stationFromRow(row)
	return &station, nil
}

func (s *Store) authenticate(ctx context.Context, secret string) (*Station, error) {
	secretHash := hashSecret(secret)
	if secretHash == "" {
		return nil, ErrUnauthorized
	}
	row, err := postgres.GetOne[stationRow](ctx, s.pool, `
UPDATE stations
SET last_seen_at = now()
WHERE secret_hash = $1
  AND enabled
	RETURNING `+stationColumnsSQL, secretHash)
	if err != nil {
		if errors.Is(err, fault.ErrNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	station := stationFromRow(row)
	return &station, nil
}

func (s *Store) observeClient(
	ctx context.Context,
	stationID int64,
	secret string,
	protocolVersion *int,
	appBuild string,
) error {
	tag, err := s.pool.Exec(ctx, `
UPDATE stations
SET last_seen_at = now(),
    protocol_version = $1,
    app_build = $2
WHERE id = $3
  AND secret_hash = $4
  AND enabled`, protocolVersion, appBuild, stationID, hashSecret(secret))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUnauthorized
	}
	return nil
}

func stationFromRow(row stationRow) Station {
	return Station{
		ID:              row.ID,
		Name:            row.Name,
		LocationID:      row.LocationID,
		Enabled:         row.Enabled,
		SecretPrefix:    row.SecretPrefix,
		Online:          row.Enabled && row.LastSeenAt != nil && row.LastSeenAt.After(time.Now().Add(-onlineWindow)),
		LastSeenAt:      row.LastSeenAt,
		AppBuild:        row.AppBuild,
		ProtocolVersion: row.ProtocolVersion,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func hashSecret(secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(hash[:])
}

func secretPrefix(secret string) (string, error) {
	const prefixLength = 14
	if len(secret) < prefixLength {
		return "", fmt.Errorf("%w: station secret is too short", fault.ErrInvalidInput)
	}
	return secret[:prefixLength], nil
}
