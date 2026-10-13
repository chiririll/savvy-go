-- name: InsertSyncGroup :execresult
INSERT INTO sync_groups (sync_categories, sync_tags, created_at) VALUES (?, ?, ?);

-- name: UpdateSyncGroupFlags :exec
UPDATE sync_groups SET sync_categories = ?, sync_tags = ? WHERE id = ?;

-- name: DeleteSyncGroup :exec
DELETE FROM sync_groups WHERE id = ?;

-- name: InsertSyncGroupSpace :exec
INSERT INTO sync_group_spaces (group_id, space_id) VALUES (?, ?);

-- name: DeleteSyncGroupSpace :execresult
DELETE FROM sync_group_spaces WHERE group_id = ? AND space_id = ?;

-- name: GetSyncGroupOfSpace :one
SELECT g.id, g.sync_categories, g.sync_tags
FROM sync_groups g JOIN sync_group_spaces s ON s.group_id = g.id
WHERE s.space_id = ?;

-- name: GetSyncGroup :one
SELECT id, sync_categories, sync_tags FROM sync_groups WHERE id = ?;

-- name: ListSyncGroupSpaces :many
SELECT space_id FROM sync_group_spaces WHERE group_id = ? ORDER BY space_id;

-- name: ListSyncGroups :many
SELECT id, sync_categories, sync_tags FROM sync_groups ORDER BY id;
