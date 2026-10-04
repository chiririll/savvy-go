-- name: CountUsers :one
SELECT COUNT(*) FROM users WHERE deleted_at IS NULL;

-- name: CountAdmins :one
SELECT COUNT(*) FROM users WHERE role = ? AND deleted_at IS NULL;

-- name: GetUser :one
SELECT id, name, email, password, role, is_sso_only,
	two_factor_secret, two_factor_enabled, two_factor_confirmed, deleted_at, created_at, updated_at
FROM users WHERE id = ?;

-- name: GetUserByEmail :one
SELECT id, name, email, password, role, is_sso_only,
	two_factor_secret, two_factor_enabled, two_factor_confirmed, deleted_at, created_at, updated_at
FROM users WHERE lower(email) = ?;

-- name: ListUsers :many
SELECT id, name, email, password, role, is_sso_only,
	two_factor_secret, two_factor_enabled, two_factor_confirmed, deleted_at, created_at, updated_at
FROM users WHERE deleted_at IS NULL ORDER BY name;

-- name: InsertUser :execresult
INSERT INTO users (name, email, password, role, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateUserPassword :exec
UPDATE users SET password = ?, updated_at = ? WHERE id = ?;

-- name: UpdateUserWithPassword :exec
UPDATE users SET name=?, email=?, role=?, password=?, updated_at=? WHERE id=?;

-- name: UpdateUserProfile :exec
UPDATE users SET name=?, email=?, role=?, updated_at=? WHERE id=?;

-- name: MarkUserSSOOnly :exec
UPDATE users SET is_sso_only=1, updated_at=? WHERE id=?;

-- name: SetUserRole :exec
UPDATE users SET role=?, updated_at=? WHERE id=?;

-- name: SetUserTwoFactor :exec
UPDATE users SET two_factor_secret=?, two_factor_enabled=?, two_factor_confirmed=?, updated_at=? WHERE id=?;

-- name: TombstoneUser :exec
-- A deleted user keeps their row (ids in space databases still resolve) but
-- loses everything that identifies them or lets them sign in.
UPDATE users SET name = 'Deleted user', email = 'deleted-' || id || '@invalid', password = NULL,
	is_sso_only = 0, two_factor_secret = NULL, two_factor_enabled = 0, two_factor_confirmed = 0,
	deleted_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: DeleteUserCredentials :exec
DELETE FROM auth_sessions WHERE user_id = ?;

-- name: DeleteUserAPITokens :exec
DELETE FROM api_tokens WHERE user_id = ?;

-- name: DeleteUserPasskeys :exec
DELETE FROM webauthn_credentials WHERE user_id = ?;

-- name: DeleteUserIdentities :exec
DELETE FROM user_identities WHERE user_id = ?;

-- name: DeleteUserPasswordTokens :exec
DELETE FROM password_tokens WHERE user_id = ?;

-- name: DeleteUserRecoveryCodes :exec
DELETE FROM two_factor_recovery_codes WHERE user_id = ?;

-- name: ListUserNames :many
SELECT id, name, deleted_at FROM users;

-- name: ListUsersWithTwoFactorSecret :many
SELECT id, two_factor_secret FROM users
WHERE two_factor_secret IS NOT NULL AND two_factor_secret != '';

-- name: UpdateUserTwoFactorSecret :exec
UPDATE users SET two_factor_secret=?, updated_at=? WHERE id=?;
