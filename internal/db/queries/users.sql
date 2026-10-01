-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CountAdmins :one
SELECT COUNT(*) FROM users WHERE role = ?;

-- name: GetUser :one
SELECT id, name, email, password, role, is_sso_only,
	two_factor_secret, two_factor_enabled, two_factor_confirmed, created_at, updated_at
FROM users WHERE id = ?;

-- name: GetUserByEmail :one
SELECT id, name, email, password, role, is_sso_only,
	two_factor_secret, two_factor_enabled, two_factor_confirmed, created_at, updated_at
FROM users WHERE lower(email) = ?;

-- name: ListUsers :many
SELECT id, name, email, password, role, is_sso_only,
	two_factor_secret, two_factor_enabled, two_factor_confirmed, created_at, updated_at
FROM users ORDER BY name;

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

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;

-- name: ListUsersWithTwoFactorSecret :many
SELECT id, two_factor_secret FROM users
WHERE two_factor_secret IS NOT NULL AND two_factor_secret != '';

-- name: UpdateUserTwoFactorSecret :exec
UPDATE users SET two_factor_secret=?, updated_at=? WHERE id=?;
