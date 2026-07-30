-- name: UpsertGroup :one
INSERT INTO groups (
  id,
  name,
  description
)
VALUES (
  $1,
  $2,
  $3
)
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  updated_at = NOW()
RETURNING
  id,
  name,
  description,
  created_at,
  updated_at;

-- name: GetGroup :one
SELECT
  g.id,
  g.name,
  g.description,
  COALESCE(member_counts.member_count, 0)::INT4 AS member_count,
  g.created_at,
  g.updated_at
FROM groups AS g
LEFT JOIN (
  SELECT group_id, COUNT(*)::INT4 AS member_count
  FROM group_memberships
  GROUP BY group_id
) AS member_counts
  ON member_counts.group_id = g.id
WHERE g.id = $1;
