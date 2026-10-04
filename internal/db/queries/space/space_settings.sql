-- name: GetSpaceSetting :one
SELECT value FROM space_settings WHERE key = ?;

-- name: UpsertSpaceSetting :exec
INSERT INTO space_settings(key, value) VALUES(?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value;

-- name: ListSpaceSettings :many
SELECT key, value FROM space_settings;
