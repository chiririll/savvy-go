-- name: InsertAPIToken :execresult
INSERT INTO api_tokens (user_id, name, token_hash, prefix, scope, expires_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListAPITokensByUser :many
SELECT id, user_id, name, token_hash, prefix, scope, expires_at, last_used_at, created_at, updated_at
FROM api_tokens WHERE user_id = ? ORDER BY id DESC;

-- name: GetAPITokenByHash :one
SELECT id, user_id, name, token_hash, prefix, scope, expires_at, last_used_at, created_at, updated_at
FROM api_tokens WHERE token_hash = ?;

-- name: TouchAPIToken :exec
UPDATE api_tokens SET last_used_at = ? WHERE id = ?;

-- name: DeleteAPIToken :execresult
DELETE FROM api_tokens WHERE id = ? AND user_id = ?;
