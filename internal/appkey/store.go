// Package appkey owns location-scoped credentials used by the companion app.
package appkey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/listing"
	"github.com/woodleighschool/woodgate/internal/postgres"
)

// ErrUnauthorized reports a missing, revoked, expired, or unknown credential.
var ErrUnauthorized = errors.New("app key unauthorized")

// Key is a machine credential without its secret or hash.
type Key struct {
	ReadUsers     bool              `json:"-" db:"read_users"`
	ReadLocations bool              `json:"-" db:"read_locations"`
	ReadBranding  bool              `json:"-" db:"read_branding"`
	ID            int64             `json:"id" db:"id"`
	AppID         uuid.UUID         `json:"app_id" db:"app_id"`
	Name          string            `json:"name" db:"name"`
	KeyPrefix     string            `json:"key_prefix" db:"key_prefix"`
	AllLocations  bool              `json:"all_locations" db:"all_locations"`
	Locations     []LocationSummary `json:"locations" db:"locations" nullable:"false"`
	ExpiresAt     *time.Time        `json:"expires_at,omitempty" db:"expires_at"`
	LastUsedAt    *time.Time        `json:"last_used_at,omitempty" db:"last_used_at"`
	CreatedAt     time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at" db:"updated_at"`
}

// Allows reports whether this key can use a location.
func (key Key) Allows(locationID int64) bool {
	return key.AllLocations || slices.ContainsFunc(key.Locations, func(location LocationSummary) bool {
		return location.ID == locationID
	})
}

// Mutation replaces a credential's name, expiry, and location scope.
type Mutation struct {
	Name         string     `json:"name" minLength:"1"`
	AllLocations bool       `json:"all_locations"`
	LocationIDs  []int64    `json:"location_ids" nullable:"false"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

// CreatedKey includes the secret returned once at creation.
type CreatedKey struct {
	Key

	APIKey string `json:"api_key"`
}

// Store persists credentials and their effective location scope.
type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

const keyColumns = `k.id,k.app_id,k.name,k.key_prefix,k.all_locations,k.read_users,k.read_locations,k.read_branding,k.expires_at,k.last_used_at,k.created_at,k.updated_at,
 COALESCE((SELECT jsonb_agg(jsonb_build_object('id',l.id,'name',l.name) ORDER BY lower(l.name),l.id) FROM app_key_locations kl JOIN locations l ON l.id=kl.location_id WHERE kl.app_key_id=k.id),'[]'::jsonb) AS locations`

func (s *Store) List(ctx context.Context, params listing.Params) ([]Key, int, error) {
	params = listing.Normalize(params)
	if err := listing.Validate(params); err != nil {
		return nil, 0, err
	}
	var where postgres.WhereBuilder
	where.Addf("k.deleted_at IS NULL")
	if params.Q != "" {
		where.Addf("k.name ILIKE '%%'||%s||'%%'", params.Q)
	}
	whereSQL, args := where.Build()
	return postgres.ListWithCount[Key](ctx, s.pool, postgres.ListQuery{
		SelectSQL: "SELECT " + keyColumns + " FROM app_keys k", WhereSQL: whereSQL, Args: args,
		OrderKeys:    map[string]postgres.OrderExpr{"id": {SQL: "k.id"}, "name": {SQL: "lower(k.name)"}, "created_at": {SQL: "k.created_at"}, "expires_at": {SQL: "k.expires_at"}, "last_used_at": {SQL: "k.last_used_at"}},
		DefaultOrder: []postgres.OrderExpr{{SQL: "k.created_at", Descending: true}, {SQL: "k.id", Descending: true}}, Params: params,
	})
}

func (s *Store) Get(ctx context.Context, id int64) (*Key, error) {
	key, err := postgres.GetOne[Key](ctx, s.pool, "SELECT "+keyColumns+" FROM app_keys k WHERE k.id=$1 AND k.deleted_at IS NULL", id)
	if err != nil {
		return nil, postgres.GetError(err)
	}
	return &key, nil
}

func (s *Store) Create(ctx context.Context, mutation Mutation) (*CreatedKey, error) {
	if err := mutation.normalize(); err != nil {
		return nil, err
	}
	secret, prefix, hash, err := generateSecret()
	if err != nil {
		return nil, err
	}
	var id int64
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `INSERT INTO app_keys(name,key_prefix,key_hash,all_locations,expires_at) VALUES($1,$2,$3,$4,$5) RETURNING id`, mutation.Name, prefix, hash, mutation.AllLocations, mutation.ExpiresAt).Scan(&id); err != nil {
			return postgres.MutationError(err)
		}
		return replaceLocations(ctx, tx, id, mutation.LocationIDs)
	})
	if err != nil {
		return nil, err
	}
	key, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &CreatedKey{Key: *key, APIKey: secret}, nil
}

func (s *Store) Update(ctx context.Context, id int64, mutation Mutation) (*Key, error) {
	if err := mutation.normalize(); err != nil {
		return nil, err
	}
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE app_keys SET name=$2,all_locations=$3,expires_at=$4,updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id, mutation.Name, mutation.AllLocations, mutation.ExpiresAt)
		if err != nil {
			return postgres.MutationError(err)
		}
		if tag.RowsAffected() == 0 {
			return fault.ErrNotFound
		}
		return replaceLocations(ctx, tx, id, mutation.LocationIDs)
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// Delete revokes the credential while retaining event attribution.
func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `UPDATE app_keys SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fault.ErrNotFound
	}
	return nil
}

// Authenticate preserves the deployed companion's secret hashing contract.
func (s *Store) Authenticate(ctx context.Context, secret string) (*Key, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, ErrUnauthorized
	}
	hash := sha256.Sum256([]byte(secret))
	key, err := postgres.GetOne[Key](ctx, s.pool, `UPDATE app_keys AS k SET last_used_at=now()
 WHERE key_hash=$1 AND deleted_at IS NULL AND (expires_at IS NULL OR expires_at>now()) RETURNING `+keyColumns, hex.EncodeToString(hash[:]))
	if errors.Is(err, fault.ErrNotFound) {
		return nil, ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func replaceLocations(ctx context.Context, tx pgx.Tx, id int64, locations []int64) error {
	if _, err := tx.Exec(ctx, "DELETE FROM app_key_locations WHERE app_key_id=$1", id); err != nil {
		return err
	}
	if len(locations) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, "INSERT INTO app_key_locations(app_key_id,location_id) SELECT $1,unnest($2::bigint[])", id, locations)
	return postgres.MutationError(err)
}

func (mutation *Mutation) normalize() error {
	mutation.Name = strings.TrimSpace(mutation.Name)
	if mutation.Name == "" {
		return fmt.Errorf("%w: name must not be empty", fault.ErrInvalidInput)
	}
	mutation.LocationIDs = slices.Clone(mutation.LocationIDs)
	slices.Sort(mutation.LocationIDs)
	mutation.LocationIDs = slices.Compact(mutation.LocationIDs)
	for _, id := range mutation.LocationIDs {
		if id < 1 {
			return fmt.Errorf("%w: location IDs must be positive", fault.ErrInvalidInput)
		}
	}
	return nil
}

func generateSecret() (string, string, string, error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", "", "", fmt.Errorf("generate app key: %w", err)
	}
	secret := "woodgate_" + base64.RawURLEncoding.EncodeToString(random)
	hash := sha256.Sum256([]byte(secret))
	return secret, secret[:12], hex.EncodeToString(hash[:]), nil
}

// LocationSummary is a location available to credential administrators.
type LocationSummary struct {
	ID   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

func (s *Store) Locations(ctx context.Context, params listing.Params) ([]LocationSummary, int, error) {
	params = listing.Normalize(params)
	if err := listing.Validate(params); err != nil {
		return nil, 0, err
	}
	var where postgres.WhereBuilder
	if params.Q != "" {
		where.Addf("name ILIKE '%%'||%s||'%%'", params.Q)
	}
	whereSQL, args := where.Build()
	return postgres.ListWithCount[LocationSummary](ctx, s.pool, postgres.ListQuery{SelectSQL: "SELECT id,name FROM locations", WhereSQL: whereSQL, Args: args,
		OrderKeys: map[string]postgres.OrderExpr{"name": {SQL: "lower(name)"}}, DefaultOrder: []postgres.OrderExpr{{SQL: "lower(name)"}, {SQL: "id"}}, Params: params})
}
