-- name: InsertSession :execresult
INSERT INTO auth_sessions (
	user_id, token_hash, csrf, ip, user_agent, remember_me,
	last_used_at, refreshed_at, idle_expires_at, absolute_expires_at,
	created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetSessionByTokenHash :one
SELECT id, user_id, token_hash, csrf, remember_me, last_used_at, refreshed_at,
	idle_expires_at, absolute_expires_at, revoked_at, created_at
FROM auth_sessions WHERE token_hash = ?;

-- name: TouchSession :exec
UPDATE auth_sessions SET last_used_at = ?, idle_expires_at = ?, updated_at = ?
WHERE id = ?;

-- name: RefreshSession :exec
UPDATE auth_sessions SET token_hash=?, csrf=?, last_used_at=?, refreshed_at=?,
	idle_expires_at=?, absolute_expires_at=?, updated_at=?
WHERE id=?;

-- name: RevokeSession :exec
UPDATE auth_sessions SET revoked_at = ?, updated_at = ? WHERE id = ?;

-- name: RevokeUserSessions :exec
UPDATE auth_sessions SET revoked_at = ?, updated_at = ?
WHERE user_id = ? AND revoked_at IS NULL;

-- name: RevokeOtherSessions :execresult
UPDATE auth_sessions SET revoked_at = ?, updated_at = ?
WHERE user_id = ? AND id != ? AND revoked_at IS NULL;
