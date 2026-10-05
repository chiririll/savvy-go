-- name: InsertIdentityProvider :execresult
INSERT INTO identity_providers (name, slug, protocol, preset, enabled, sort_order, config, secrets,
	claim_mappings, role_mapping, default_role, allow_jit, sync_role_on_login, link_by_email, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?);

-- name: UpdateIdentityProvider :exec
UPDATE identity_providers SET name=?, slug=?, protocol=?, preset=?, enabled=?, sort_order=?, config=?, secrets=?,
	claim_mappings=?, role_mapping=?, default_role=?, allow_jit=?, sync_role_on_login=?, link_by_email=?, updated_at=?
WHERE id=?;

-- name: CountSSOOnlyOrphans :one
-- One bind: sqlc sqlite mishandles the same named arg twice in EXISTS/NOT EXISTS.
SELECT COUNT(*) FROM (
  SELECT u.id
  FROM users u
  INNER JOIN user_identities mine
    ON mine.user_id = u.id AND mine.identity_provider_id = sqlc.arg(identity_provider_id)
  LEFT JOIN user_identities other
    ON other.user_id = u.id AND other.identity_provider_id <> mine.identity_provider_id
  WHERE u.is_sso_only = 1
  GROUP BY u.id
  HAVING COUNT(other.id) = 0
);

-- name: DeleteIdentityProvider :exec
DELETE FROM identity_providers WHERE id = ?;

-- name: InsertSSOTicket :exec
INSERT INTO sso_login_tickets (ticket, user_id, requires_2fa, expires_at, created_at, updated_at)
VALUES (?,?,?,?,?,?);

-- name: ConsumeSSOTicket :execresult
UPDATE sso_login_tickets SET consumed_at=?, updated_at=?
WHERE ticket=? AND consumed_at IS NULL AND expires_at > ?;

-- name: GetSSOTicket :one
SELECT user_id, requires_2fa FROM sso_login_tickets WHERE ticket=?;

-- name: InsertSSOState :exec
INSERT INTO sso_login_states (state, identity_provider_id, nonce, code_verifier, saml_request_id, redirect_after, expires_at, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?);

-- name: AttachSAMLRequestID :exec
UPDATE sso_login_states SET saml_request_id=? WHERE state=?;

-- name: GetSSOState :one
SELECT identity_provider_id, nonce, code_verifier, saml_request_id, redirect_after, expires_at
FROM sso_login_states WHERE state=?;

-- name: DeleteSSOState :exec
DELETE FROM sso_login_states WHERE state=?;

-- name: ListIdentityProviders :many
SELECT id, name, slug, protocol, preset, enabled, sort_order, config, secrets,
	claim_mappings, role_mapping, default_role, allow_jit, sync_role_on_login, link_by_email, created_at, updated_at
FROM identity_providers
WHERE id = COALESCE(sqlc.narg('id'), id)
  AND slug = COALESCE(sqlc.narg('slug'), slug)
  AND CASE WHEN sqlc.narg('enabled_only') IS NULL THEN 1 ELSE enabled END = 1
ORDER BY sort_order, id;

-- name: CountEnabledIdentityProviders :one
SELECT COUNT(*) FROM identity_providers WHERE enabled = 1;

-- name: GetUserIdentity :one
SELECT id, user_id FROM user_identities WHERE identity_provider_id=? AND subject=?;

-- name: TouchUserIdentity :exec
UPDATE user_identities SET last_login_at=?, claims=?, updated_at=? WHERE id=?;

-- name: InsertUserIdentity :exec
INSERT INTO user_identities (user_id, identity_provider_id, subject, last_login_at, claims, created_at, updated_at)
VALUES (?,?,?,?,?,?,?);
