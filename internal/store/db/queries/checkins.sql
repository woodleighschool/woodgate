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
