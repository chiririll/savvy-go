-- name: InsertTwoFactorChallenge :exec
INSERT INTO two_factor_challenges (user_id, token_hash, expires_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetValidChallengeUserID :one
SELECT user_id FROM two_factor_challenges
WHERE token_hash = ? AND consumed_at IS NULL AND expires_at > ?;

-- name: ConsumeTwoFactorChallenge :execresult
UPDATE two_factor_challenges SET consumed_at = ?, updated_at = ?
WHERE token_hash = ? AND consumed_at IS NULL AND expires_at > ?;

-- name: GetChallengeUserID :one
SELECT user_id FROM two_factor_challenges WHERE token_hash = ?;

-- name: CountTwoFactorChallenges :one
SELECT COUNT(*) FROM two_factor_challenges;

-- name: FailTwoFactorChallenge :one
UPDATE two_factor_challenges
SET attempts = attempts + 1,
    consumed_at = CASE WHEN attempts + 1 >= sqlc.arg(max_attempts) THEN sqlc.arg(consumed_now) ELSE consumed_at END,
    updated_at = sqlc.arg(updated_now)
WHERE token_hash = sqlc.arg(token_hash) AND consumed_at IS NULL AND expires_at > sqlc.arg(valid_at)
RETURNING user_id, attempts;

-- name: GetTwoFactorLock :one
SELECT locked_until FROM two_factor_failures WHERE user_id = ?;

-- name: AddTwoFactorFailure :one
INSERT INTO two_factor_failures (user_id, failures) VALUES (?, 1)
ON CONFLICT (user_id) DO UPDATE SET failures = failures + 1
RETURNING failures;

-- name: LockTwoFactor :exec
UPDATE two_factor_failures SET locked_until = ? WHERE user_id = ?;

-- name: ResetTwoFactorFailures :exec
DELETE FROM two_factor_failures WHERE user_id = ?;
