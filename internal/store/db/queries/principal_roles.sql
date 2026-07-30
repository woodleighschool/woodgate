-- name: CreatePrincipalRole :exec
INSERT INTO principal_roles (
  id,
  principal_kind,
  principal_id,
  role
)
VALUES (
  $1,
  $2,
  $3,
  $4
);

-- name: ListPrincipalRolesByPrincipals :many
SELECT
  id,
  principal_kind,
  principal_id,
  role,
  created_at,
  updated_at
FROM principal_roles
WHERE principal_kind = sqlc.arg(principal_kind)
  AND principal_id = ANY(sqlc.arg(principal_ids)::UUID[])
ORDER BY principal_id, role;

-- name: ListPrincipalRoles :many
SELECT
  id,
  principal_kind,
  principal_id,
  role,
  created_at,
  updated_at
FROM principal_roles
ORDER BY principal_kind, principal_id, role;

-- name: DeletePrincipalRolesByPrincipal :exec
DELETE FROM principal_roles
WHERE principal_kind = $1
  AND principal_id = $2;
