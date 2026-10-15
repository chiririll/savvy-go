-- name: InsertAPIToken :execresult
INSERT INTO api_tokens (user_id, name, token_hash, prefix, kind, permissions, all_spaces, expires_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListAPITokensByUser :many
SELECT id, user_id, name, token_hash, prefix, kind, permissions, all_spaces, expires_at, last_used_at, created_at, updated_at
FROM api_tokens WHERE user_id = ? ORDER BY id DESC;

-- name: GetAPITokenByHash :one
SELECT id, user_id, name, token_hash, prefix, kind, permissions, all_spaces, expires_at, last_used_at, created_at, updated_at
FROM api_tokens WHERE token_hash = ?;

-- name: TouchAPIToken :exec
UPDATE api_tokens SET last_used_at = ? WHERE id = ?;

-- name: DeleteAPIToken :execresult
DELETE FROM api_tokens WHERE id = ? AND user_id = ?;

-- name: InsertAPITokenSpace :exec
INSERT INTO api_token_spaces (api_token_id, space_id) VALUES (?, ?);

-- name: ListAPITokenSpaces :many
SELECT space_id FROM api_token_spaces WHERE api_token_id = ? ORDER BY space_id;

-- name: ListAPITokenSpacesByUser :many
SELECT s.api_token_id, s.space_id
FROM api_token_spaces s JOIN api_tokens t ON t.id = s.api_token_id
WHERE t.user_id = ? ORDER BY s.api_token_id, s.space_id;

-- name: UpdateAPITokenGrant :execresult
UPDATE api_tokens SET permissions = ?, all_spaces = ?, updated_at = ? WHERE id = ? AND user_id = ?;

-- name: DeleteAPITokenSpaces :exec
DELETE FROM api_token_spaces WHERE api_token_id = ?;
