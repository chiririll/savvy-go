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
