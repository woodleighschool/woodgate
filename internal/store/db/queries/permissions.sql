-- name: CreatePermission :exec
INSERT INTO permissions (
  id,
  subject_kind,
  subject_id,
  resource,
  action,
  location_id,
  asset_type
)
VALUES (
  $1,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7
);

-- name: ListPermissionsBySubjects :many
SELECT
  id,
  subject_kind,
  subject_id,
  resource,
  action,
  location_id,
  asset_type,
  created_at,
  updated_at
FROM permissions
WHERE subject_kind = sqlc.arg(subject_kind)
  AND subject_id = ANY(sqlc.arg(subject_ids)::UUID[])
ORDER BY subject_id, resource, location_id NULLS FIRST, asset_type NULLS FIRST;

-- name: ListPermissions :many
SELECT
  id,
  subject_kind,
  subject_id,
  resource,
  action,
  location_id,
  asset_type,
  created_at,
  updated_at
FROM permissions
ORDER BY subject_kind, subject_id, resource, action, location_id NULLS FIRST, asset_type NULLS FIRST;

-- name: DeletePermissionsBySubject :exec
DELETE FROM permissions
WHERE subject_kind = $1
  AND subject_id = $2;
