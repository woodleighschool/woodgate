-- name: CreateAPIKey :one
INSERT INTO api_keys (
  id,
  name,
  key_prefix,
  key_hash,
  expires_at
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
  key_prefix,
  key_hash,
  last_used_at,
  expires_at,
  created_at;

-- name: GetAPIKey :one
SELECT
  id,
  name,
  key_prefix,
  key_hash,
  last_used_at,
  expires_at,
  created_at
FROM api_keys
WHERE id = $1;

-- name: GetAPIKeyByHash :one
SELECT
  id,
  name,
  key_prefix,
  key_hash,
  last_used_at,
  expires_at,
  created_at
FROM api_keys
WHERE key_hash = $1;

-- name: DeleteAPIKey :one
DELETE FROM api_keys
WHERE id = $1
RETURNING id;

-- name: UpdateAPIKey :one
UPDATE api_keys
SET
  name = $2,
  expires_at = $3
WHERE id = $1
RETURNING
  id,
  name,
  key_prefix,
  key_hash,
  last_used_at,
  expires_at,
  created_at;

-- name: UpdateAPIKeyLastUsedAt :exec
UPDATE api_keys
SET last_used_at = NOW()
WHERE id = $1;
