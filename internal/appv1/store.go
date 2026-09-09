// Package appv1 preserves the installed companion's UUID-based HTTP contract.
package appv1

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/woodgate/internal/appkey"
	"github.com/woodleighschool/woodgate/internal/postgres"
)

// Store projects the shared domain into companion identities.
type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

type location struct {
	InternalID        int64       `json:"-" db:"internal_id"`
	ID                uuid.UUID   `json:"id" db:"id"`
	Name              string      `json:"name" db:"name"`
	Description       string      `json:"description" db:"description"`
	Enabled           bool        `json:"enabled" db:"enabled"`
	Notes             bool        `json:"notes" db:"notes"`
	Photo             bool        `json:"photo" db:"photo"`
	BackgroundAssetID *uuid.UUID  `json:"background_asset_id,omitempty" db:"background_asset_id"`
	LogoAssetID       *uuid.UUID  `json:"logo_asset_id,omitempty" db:"logo_asset_id"`
	GroupIDs          []uuid.UUID `json:"group_ids" db:"group_ids"`
	CreatedAt         time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at" db:"updated_at"`
}

const locationColumns = `l.id AS internal_id,l.app_id AS id,l.name,l.description,l.enabled,l.notes,l.photo,
 l.background_app_id AS background_asset_id,l.logo_app_id AS logo_asset_id,l.created_at,l.updated_at,
 COALESCE((SELECT array_agg(g.app_id ORDER BY g.id) FROM location_directory_groups lg JOIN directory_groups g ON g.id=lg.group_id WHERE lg.location_id=l.id),'{}'::uuid[]) AS group_ids`

func (s *Store) locations(ctx context.Context, key appkey.Key) ([]location, error) {
	return postgres.GetAll[location](ctx, s.pool, "SELECT "+locationColumns+" FROM locations l WHERE ($1 OR EXISTS (SELECT 1 FROM app_key_locations kl WHERE kl.app_key_id=$2 AND kl.location_id=l.id)) ORDER BY l.name,l.id", key.AllLocations, key.ID)
}
func (s *Store) location(ctx context.Context, id uuid.UUID) (*location, error) {
	item, err := postgres.GetOne[location](ctx, s.pool, "SELECT "+locationColumns+" FROM locations l WHERE l.app_id=$1", id)
	if err != nil {
		return nil, postgres.GetError(err)
	}
	return &item, nil
}

type person struct {
	InternalID  int64     `json:"-" db:"internal_id"`
	ID          uuid.UUID `json:"id" db:"id"`
	UPN         string    `json:"upn" db:"upn"`
	DisplayName string    `json:"display_name" db:"display_name"`
	Department  string    `json:"department" db:"department"`
	Source      string    `json:"source" db:"source"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

const personColumns = `u.id AS internal_id,u.app_id AS id,COALESCE(u.user_principal_name,u.email,'') AS upn,u.name AS display_name,
 COALESCE(u.department,'') AS department,u.source::text AS source,u.created_at,u.updated_at`

func (s *Store) people(ctx context.Context, locationID int64, limit, offset int) ([]person, int, error) {
	const where = ` FROM users u WHERE u.deleted_at IS NULL AND
 EXISTS(SELECT 1 FROM location_directory_groups lg JOIN directory_group_memberships gm ON gm.group_id=lg.group_id WHERE lg.location_id=$1 AND gm.user_id=u.id)`
	var count int
	if err := s.pool.QueryRow(ctx, "SELECT count(*)"+where, locationID).Scan(&count); err != nil {
		return nil, 0, err
	}
	items, err := postgres.GetAll[person](ctx, s.pool, "SELECT "+personColumns+where+" ORDER BY u.name,u.id LIMIT NULLIF($2,0) OFFSET $3", locationID, limit, offset)
	return items, count, err
}
func (s *Store) person(ctx context.Context, id uuid.UUID) (*person, error) {
	item, err := postgres.GetOne[person](ctx, s.pool, "SELECT "+personColumns+" FROM users u WHERE u.app_id=$1", id)
	if err != nil {
		return nil, postgres.GetError(err)
	}
	return &item, nil
}

type attachment struct {
	LocationID int64 `db:"location_id"`
	ObjectID   int64 `db:"object_id"`
}

func (s *Store) attachment(ctx context.Context, id uuid.UUID) (*attachment, error) {
	item, err := postgres.GetOne[attachment](ctx, s.pool, `SELECT id AS location_id,
 CASE WHEN background_app_id=$1 THEN background_object_id ELSE logo_object_id END AS object_id
 FROM locations WHERE background_app_id=$1 OR logo_app_id=$1`, id)
	if err != nil {
		return nil, postgres.GetError(err)
	}
	return &item, nil
}
