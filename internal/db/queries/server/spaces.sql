-- name: InsertSpace :one
INSERT INTO spaces (uuid, name, created_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
RETURNING id;

-- name: DeleteSpaceRow :exec
DELETE FROM spaces WHERE id = ?;

-- name: GetSpace :one
SELECT id, uuid, name, created_by, created_at, updated_at FROM spaces WHERE id = ?;

-- name: ListSpaceIDs :many
SELECT id FROM spaces ORDER BY id;

-- name: ListUserSpaces :many
SELECT s.id, s.uuid, s.name, m.role
FROM space_members m JOIN spaces s ON s.id = m.space_id
WHERE m.user_id = ?
ORDER BY s.id;

-- name: GetMemberRole :one
SELECT role FROM space_members WHERE space_id = ? AND user_id = ?;

-- name: UpsertSpaceMember :exec
INSERT INTO space_members (space_id, user_id, role, created_at) VALUES (?, ?, ?, ?)
ON CONFLICT (space_id, user_id) DO UPDATE SET role = excluded.role;
