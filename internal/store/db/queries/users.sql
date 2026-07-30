-- name: UpsertUser :one
INSERT INTO users (
  id,
  upn,
  display_name,
  department,
  source
)
VALUES (
  $1,
  $2,
  $3,
  $4,
  $5
)
ON CONFLICT (id) DO UPDATE SET
  upn = EXCLUDED.upn,
  display_name = EXCLUDED.display_name,
  department = EXCLUDED.department,
  source = EXCLUDED.source,
  updated_at = NOW()
RETURNING
  id,
  upn,
  display_name,
  department,
  source,
  created_at,
  updated_at;

-- name: GetUser :one
SELECT
  id,
  upn,
  display_name,
  department,
  source,
  created_at,
  updated_at
FROM users
WHERE id = $1;

-- name: GetUserByUPN :one
SELECT
  id,
  upn,
  display_name,
  department,
  source,
  created_at,
  updated_at
FROM users
WHERE lower(upn) = lower($1);
