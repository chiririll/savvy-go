-- name: DeleteUserEmailVerificationTokens :exec
DELETE FROM email_verification_tokens WHERE user_id = ?;

-- name: InsertEmailVerificationToken :exec
INSERT INTO email_verification_tokens (user_id, email, token_hash, expires_at, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetEmailVerificationToken :one
SELECT id, user_id, email, expires_at FROM email_verification_tokens WHERE token_hash = ?;

-- name: ConsumeEmailVerificationToken :execresult
DELETE FROM email_verification_tokens WHERE token_hash = ?;
