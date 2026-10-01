-- name: InsertPasswordToken :exec
INSERT INTO password_tokens (user_id, token_hash, expires_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: ConsumePasswordToken :execresult
UPDATE password_tokens SET consumed_at = ?, updated_at = ?
WHERE token_hash = ? AND consumed_at IS NULL AND expires_at > ?;

-- name: RevokeActivePasswordTokens :exec
UPDATE password_tokens SET consumed_at = ?, updated_at = ?
WHERE user_id = ? AND consumed_at IS NULL;

-- name: GetValidPasswordToken :one
SELECT id, user_id, expires_at FROM password_tokens
WHERE token_hash = ? AND consumed_at IS NULL AND expires_at > ?;

-- name: GetPasswordTokenByHash :one
SELECT id, user_id, expires_at FROM password_tokens WHERE token_hash = ?;
