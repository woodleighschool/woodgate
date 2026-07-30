-- name: CreateLocation :one
INSERT INTO locations (
  id,
  name,
  description,
  enabled,
  notes,
  photo,
  background_asset_id,
  logo_asset_id,
  group_ids
)
VALUES (
  $1,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  $9
)
RETURNING
  id,
  name,
  description,
  enabled,
  notes,
  photo,
  background_asset_id,
  logo_asset_id,
  group_ids,
  created_at,
  updated_at;

-- name: GetLocation :one
SELECT
  id,
  name,
  description,
  enabled,
  notes,
  photo,
  background_asset_id,
  logo_asset_id,
  group_ids,
  created_at,
  updated_at
FROM locations
WHERE id = $1;

-- name: UpdateLocation :one
UPDATE locations
SET
  name = $2,
  description = $3,
  enabled = $4,
  notes = $5,
  photo = $6,
  background_asset_id = $7,
  logo_asset_id = $8,
  group_ids = $9,
  updated_at = NOW()
WHERE id = $1
RETURNING
  id,
  name,
  description,
  enabled,
  notes,
  photo,
  background_asset_id,
  logo_asset_id,
  group_ids,
  created_at,
  updated_at;

-- name: DeleteLocation :one
DELETE FROM locations
WHERE id = $1
RETURNING id;
