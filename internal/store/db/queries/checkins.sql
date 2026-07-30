-- name: CreateCheckin :one
INSERT INTO checkins (
  id,
  user_id,
  location_id,
  direction,
  notes,
  asset_id,
  created_by_kind,
  created_by_id
)
VALUES (
  $1,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8
)
RETURNING
  id,
  user_id,
  location_id,
  direction,
  notes,
  asset_id,
  created_by_kind,
  created_by_id,
  created_at;

-- name: GetCheckin :one
SELECT
  c.id,
  c.user_id,
  c.location_id,
  c.direction,
  c.notes,
  c.asset_id,
  c.created_by_kind,
  c.created_by_id,
  c.created_at
FROM checkins AS c
WHERE c.id = $1;
