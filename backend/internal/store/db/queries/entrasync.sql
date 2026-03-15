-- name: ConvertMissingEntraUsersToLocal :exec
UPDATE users
SET
  source = 'local',
  updated_at = NOW()
WHERE source = 'entra'
  AND NOT (id = ANY(sqlc.arg(user_ids)::UUID[]));

-- name: DeleteMissingGroups :exec
DELETE FROM groups
WHERE NOT (id = ANY(sqlc.arg(group_ids)::UUID[]));

-- name: DeleteAllGroupMemberships :exec
DELETE FROM group_memberships;
