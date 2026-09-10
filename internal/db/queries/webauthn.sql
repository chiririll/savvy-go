-- name: ListWebAuthnCredentials :many
SELECT id, name, aaguid, last_used_at, created_at
FROM webauthn_credentials WHERE user_id=? ORDER BY id DESC;

-- name: CountWebAuthnCredentials :one
SELECT COUNT(*) FROM webauthn_credentials WHERE user_id=?;

-- name: InsertWebAuthnCredential :execresult
INSERT INTO webauthn_credentials (user_id, credential_id, name, aaguid, record, counter, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?);

-- name: RenameWebAuthnCredential :execresult
UPDATE webauthn_credentials SET name=?, updated_at=? WHERE id=? AND user_id=?;

-- name: DeleteWebAuthnCredential :execresult
DELETE FROM webauthn_credentials WHERE id=? AND user_id=?;

-- name: InsertWebAuthnChallenge :exec
INSERT INTO webauthn_challenges (user_id, token_hash, type, options, expires_at, created_at, updated_at)
VALUES (?,?,?,?,?,?,?);

-- name: GetOpenWebAuthnChallenge :one
SELECT user_id, options FROM webauthn_challenges
WHERE token_hash=? AND type=? AND consumed_at IS NULL AND expires_at > ?;

-- name: ConsumeWebAuthnChallenge :exec
UPDATE webauthn_challenges SET consumed_at=?, updated_at=? WHERE token_hash=?;

-- name: ListWebAuthnRecords :many
SELECT record FROM webauthn_credentials WHERE user_id=?;

-- name: GetWebAuthnUserIDByCredential :one
SELECT user_id FROM webauthn_credentials WHERE credential_id=?;

-- name: TouchWebAuthnCredential :exec
UPDATE webauthn_credentials SET last_used_at=?, updated_at=? WHERE credential_id=?;
