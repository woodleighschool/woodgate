-- name: CreateAsset :one
INSERT INTO assets (
  id,
  name,
  type,
  content_type,
  file_extension
)
VALUES (
  $1,
  $2,
  $3,
  $4,
  $5
)
RETURNING
  id,
  name,
  type,
  content_type,
  file_extension,
  created_at,
  updated_at;

-- name: GetAsset :one
SELECT
  id,
  name,
  type,
  content_type,
  file_extension,
  created_at,
  updated_at
FROM assets
WHERE id = $1;

-- name: UpdateAsset :one
UPDATE assets
SET
  name = $2,
  content_type = $3,
  file_extension = $4,
  updated_at = NOW()
WHERE id = $1
RETURNING
  id,
  name,
  type,
  content_type,
  file_extension,
  created_at,
  updated_at;

-- name: DeleteAsset :one
DELETE FROM assets
WHERE id = $1
RETURNING id;

-- name: ListCheckinLocationIDsByAsset :many
SELECT location_id
FROM checkins
WHERE asset_id = $1;
