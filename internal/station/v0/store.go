package v0

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodgate/internal/station"
)

type repository interface {
	authenticate(context.Context, string) (*station.Station, uuid.UUID, error)
	legacyID(context.Context, string, int64) (uuid.UUID, error)
	objectID(context.Context, string, uuid.UUID) (int64, error)
	ensureLegacyID(context.Context, string, int64) (uuid.UUID, error)
}

type store struct{ pool *pgxpool.Pool }

func newStore(pool *pgxpool.Pool) *store { return &store{pool: pool} }

func (s *store) authenticate(ctx context.Context, secret string) (*station.Station, uuid.UUID, error) {
	hash := hashSecret(secret)
	if hash == "" {
		return nil, uuid.Nil, station.ErrUnauthorized
	}
	var item station.Station
	var legacyID uuid.UUID
	err := s.pool.QueryRow(ctx, `
UPDATE stations AS s
SET last_seen_at = now(), protocol_version = 0, app_build = 'legacy-v0'
FROM station_v0_mappings AS m
WHERE s.secret_hash = $1
  AND s.enabled
  AND m.kind = 'api_key'
  AND m.object_id = s.id
RETURNING s.id, s.name, s.location_id, s.enabled, s.secret_prefix,
          s.last_seen_at, s.app_build, s.protocol_version,
          s.created_at, s.updated_at, m.legacy_id`, hash).Scan(
		&item.ID, &item.Name, &item.LocationID, &item.Enabled, &item.SecretPrefix,
		&item.LastSeenAt, &item.AppBuild, &item.ProtocolVersion,
		&item.CreatedAt, &item.UpdatedAt, &legacyID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, uuid.Nil, station.ErrUnauthorized
	}
	if err != nil {
		return nil, uuid.Nil, err
	}
	item.Online = true
	return &item, legacyID, nil
}

func (s *store) legacyID(ctx context.Context, kind string, objectID int64) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT legacy_id FROM station_v0_mappings WHERE kind=$1 AND object_id=$2`, kind, objectID).Scan(&id)
	return id, err
}

func (s *store) objectID(ctx context.Context, kind string, legacyID uuid.UUID) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `SELECT object_id FROM station_v0_mappings WHERE kind=$1 AND legacy_id=$2`, kind, legacyID).Scan(&id)
	return id, err
}

func (s *store) ensureLegacyID(ctx context.Context, kind string, objectID int64) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
INSERT INTO station_v0_mappings (kind, legacy_id, object_id)
VALUES ($1, gen_random_uuid(), $2)
ON CONFLICT (kind, object_id) DO UPDATE SET object_id = EXCLUDED.object_id
RETURNING legacy_id`, kind, objectID).Scan(&id)
	return id, err
}

func hashSecret(secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(hash[:])
}
