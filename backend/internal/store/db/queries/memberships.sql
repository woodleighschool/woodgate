-- name: CreateGroupMembership :one
INSERT INTO group_memberships (
  id,
  group_id,
  user_id
)
VALUES (
  $1,
  $2,
  $3
)
RETURNING
  id,
  group_id,
  user_id,
  created_at,
  updated_at;

-- name: AddSyncedGroupMembership :exec
INSERT INTO group_memberships (
  id,
  group_id,
  user_id
)
VALUES (
  $1,
  $2,
  $3
)
ON CONFLICT (group_id, user_id) DO UPDATE SET
  updated_at = NOW();

-- name: GetGroupMembership :one
SELECT
  gm.id,
  gm.group_id,
  gm.user_id,
  gm.created_at,
  gm.updated_at
FROM group_memberships AS gm
WHERE gm.id = $1;
