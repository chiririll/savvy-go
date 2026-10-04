-- name: DeleteRecoveryCodes :exec
DELETE FROM two_factor_recovery_codes WHERE user_id = ?;

-- name: ListUnusedRecoveryCodes :many
SELECT id, code FROM two_factor_recovery_codes WHERE user_id = ? AND used_at IS NULL;

-- name: MarkRecoveryCodeUsed :exec
UPDATE two_factor_recovery_codes SET used_at=? WHERE id=?;

-- name: CountUnusedRecoveryCodes :one
SELECT COUNT(*) FROM two_factor_recovery_codes WHERE user_id = ? AND used_at IS NULL;

-- name: InvalidateUnusedRecoveryCodes :exec
UPDATE two_factor_recovery_codes SET used_at=? WHERE user_id=? AND used_at IS NULL;

-- name: InsertRecoveryCode :exec
INSERT INTO two_factor_recovery_codes (user_id, code, created_at) VALUES (?,?,?);
