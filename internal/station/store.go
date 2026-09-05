package station

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/postgres"
)

const (
	stationSessionTTL   = 15 * time.Second
	stationRejectionTTL = 5 * time.Minute
)

// ErrSessionInvalid means a control connection no longer owns the Station session.
var ErrSessionInvalid = errors.New("station session is no longer current")

// Store persists stations and their current companion connection state.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns a PostgreSQL-backed station store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

type stationRow struct {
	ID              int64     `db:"id"`
	Name            string    `db:"name"`
	LocationID      int64     `db:"location_id"`
	LocationName    string    `db:"location_name"`
	Enabled         bool      `db:"enabled"`
	State           string    `db:"state"`
	Version         string    `db:"version"`
	ProtocolVersion *int      `db:"protocol_version"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

const stationColumnsSQL = `s.id, s.name, s.location_id, l.name AS location_name, s.enabled,
    CASE
        WHEN session.station_id IS NOT NULL THEN 'online'
        WHEN rejection.station_id IS NOT NULL THEN 'incompatible'
        ELSE 'offline'
    END AS state,
    COALESCE(session.version, rejection.version, '') AS version,
    COALESCE(session.protocol_version, rejection.protocol_version) AS protocol_version,
    s.created_at, s.updated_at`

const stationSelectSQL = `SELECT ` + stationColumnsSQL + `
FROM stations AS s
JOIN locations AS l ON l.id = s.location_id
LEFT JOIN station_sessions AS session
    ON session.station_id = s.id AND session.expires_at > now()
LEFT JOIN station_rejections AS rejection
    ON rejection.station_id = s.id AND rejection.expires_at > now()`

func (s *Store) List(ctx context.Context, params StationListParams) ([]Station, int, error) {
	where := postgres.WhereBuilder{}
	if params.ListParams.Q != "" {
		where.Addf("(s.name ILIKE '%%' || %s || '%%' OR l.name ILIKE '%%' || %s || '%%')",
			params.ListParams.Q, params.ListParams.Q)
	}
	if params.LocationID > 0 {
		where.Addf("s.location_id = %s", params.LocationID)
	}
	if params.Enabled != nil {
		where.Addf("s.enabled = %s", *params.Enabled)
	}
	whereSQL, args := where.Build()
	rows, count, err := postgres.ListWithCount[stationRow](ctx, s.pool, postgres.ListQuery{
		SelectSQL: stationSelectSQL,
		WhereSQL:  whereSQL,
		Args:      args,
		OrderKeys: map[string]postgres.OrderExpr{
			"id":            {SQL: "s.id"},
			"name":          {SQL: "lower(s.name)"},
			"location.name": {SQL: "lower(l.name)"},
			"enabled":       {SQL: "s.enabled"},
			"version":       {SQL: "lower(COALESCE(session.version, rejection.version, ''))"},
			"created_at":    {SQL: "s.created_at"},
			"updated_at":    {SQL: "s.updated_at"},
		},
		DefaultOrder: []postgres.OrderExpr{{SQL: "lower(s.name)"}, {SQL: "s.id"}},
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
	row, err := postgres.GetOne[stationRow](ctx, s.pool, stationSelectSQL+"\nWHERE s.id = $1", id)
	if err != nil {
		return nil, postgres.GetError(err)
	}
	station := stationFromRow(row)
	return &station, nil
}

func (s *Store) create(ctx context.Context, mutation StationMutation, secretHash string) (*Station, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
INSERT INTO stations (name, location_id, enabled, secret_hash)
VALUES ($1, $2, $3, $4)
RETURNING id`, mutation.Name, mutation.LocationID, mutation.Enabled, secretHash).Scan(&id)
	if err != nil {
		return nil, postgres.MutationError(err)
	}
	return s.Get(ctx, id)
}

func (s *Store) update(ctx context.Context, id int64, mutation StationMutation) (*Station, error) {
	var updatedID int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
UPDATE stations
SET name = $1,
    location_id = $2,
    enabled = $3,
    updated_at = now()
WHERE id = $4
RETURNING id`, mutation.Name, mutation.LocationID, mutation.Enabled, id).Scan(&updatedID); err != nil {
			return err
		}
		if !mutation.Enabled {
			return clearClientStates(ctx, tx, id)
		}
		return nil
	})
	if err != nil {
		return nil, postgres.MutationError(err)
	}
	return s.Get(ctx, updatedID)
}

func (s *Store) delete(ctx context.Context, id int64) error {
	var deletedID int64
	err := s.pool.QueryRow(ctx, `DELETE FROM stations WHERE id = $1 RETURNING id`, id).Scan(&deletedID)
	return postgres.DeleteConflict(err, "station is still referenced")
}

func (s *Store) rotateKey(ctx context.Context, id int64, secretHash string) (*Station, error) {
	var updatedID int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
UPDATE stations
SET secret_hash = $1,
    updated_at = now()
WHERE id = $2
RETURNING id`, secretHash, id).Scan(&updatedID); err != nil {
			return err
		}
		return clearClientStates(ctx, tx, id)
	})
	if err != nil {
		return nil, postgres.MutationError(err)
	}
	return s.Get(ctx, updatedID)
}

func (s *Store) authenticate(ctx context.Context, secret string) (*Station, error) {
	secretHash := hashSecret(secret)
	if secretHash == "" {
		return nil, ErrUnauthorized
	}
	row, err := postgres.GetOne[stationRow](ctx, s.pool, stationSelectSQL+`
WHERE s.secret_hash = $1
  AND s.enabled`, secretHash)
	if err != nil {
		if errors.Is(err, fault.ErrNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	station := stationFromRow(row)
	return &station, nil
}

// ObserveRejectedClient records the latest authenticated client that could not negotiate v1.
func (s *Store) ObserveRejectedClient(
	ctx context.Context,
	stationID int64,
	secret string,
	protocolVersion *int,
	version string,
) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := lockStationCredential(ctx, tx, stationID, secret); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
INSERT INTO station_rejections (station_id, protocol_version, version, expires_at)
VALUES ($1, $2, $3, now() + $4 * interval '1 microsecond')
ON CONFLICT (station_id) DO UPDATE
SET protocol_version = EXCLUDED.protocol_version,
    version = EXCLUDED.version,
    expires_at = EXCLUDED.expires_at`,
			stationID, protocolVersion, version, stationRejectionTTL.Microseconds())
		return err
	})
}

func (s *Store) claimClientSession(
	ctx context.Context,
	stationID int64,
	secret, connectionID string,
	protocolVersion int,
	version string,
) error {
	if connectionID == "" {
		return fmt.Errorf("%w: connection ID is required", fault.ErrInvalidInput)
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := lockStationCredential(ctx, tx, stationID, secret); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO station_sessions (station_id, connection_id, protocol_version, version, expires_at)
VALUES ($1, $2, $3, $4, now() + $5 * interval '1 microsecond')
ON CONFLICT (station_id) DO UPDATE
SET connection_id = EXCLUDED.connection_id,
    protocol_version = EXCLUDED.protocol_version,
    version = EXCLUDED.version,
    expires_at = EXCLUDED.expires_at`,
			stationID, connectionID, protocolVersion, version, stationSessionTTL.Microseconds()); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM station_rejections WHERE station_id = $1`, stationID)
		return err
	})
}

func (s *Store) renewClientSession(ctx context.Context, stationID int64, connectionID string) error {
	tag, err := s.pool.Exec(ctx, `
UPDATE station_sessions AS session
SET expires_at = now() + $3 * interval '1 microsecond'
FROM stations AS station
WHERE session.station_id = $1
  AND session.connection_id = $2
  AND session.expires_at > now()
  AND station.id = session.station_id
  AND station.enabled`, stationID, connectionID, stationSessionTTL.Microseconds())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrSessionInvalid
	}
	return nil
}

func (s *Store) releaseClientSession(ctx context.Context, stationID int64, connectionID string) error {
	_, err := s.pool.Exec(ctx, `
DELETE FROM station_sessions
WHERE station_id = $1 AND connection_id = $2`, stationID, connectionID)
	return err
}

func lockStationCredential(ctx context.Context, tx pgx.Tx, stationID int64, secret string) error {
	var found bool
	err := tx.QueryRow(ctx, `
SELECT true
FROM stations
WHERE id = $1 AND enabled AND secret_hash = $2
FOR SHARE`, stationID, hashSecret(secret)).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrSessionInvalid
	}
	return err
}

func clearClientStates(ctx context.Context, tx pgx.Tx, stationID int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM station_sessions WHERE station_id = $1`, stationID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `DELETE FROM station_rejections WHERE station_id = $1`, stationID)
	return err
}

func stationFromRow(row stationRow) Station {
	return Station{
		ID:              row.ID,
		Name:            row.Name,
		LocationID:      row.LocationID,
		Location:        Location{ID: row.LocationID, Name: row.LocationName},
		Enabled:         row.Enabled,
		State:           State(row.State),
		Version:         row.Version,
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
