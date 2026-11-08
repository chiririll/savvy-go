-- name: InsertOAuthClient :exec
INSERT INTO oauth_clients (client_id, name, redirect_uris, ip, user_agent, origin, created_at) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: InsertManualOAuthClient :exec
INSERT INTO oauth_clients (client_id, name, redirect_uris, created_at, kind, secret_hash)
VALUES (?, ?, ?, ?, 'manual', ?);

-- name: UpsertCIMDOAuthClient :exec
INSERT INTO oauth_clients (client_id, name, redirect_uris, created_at, kind, expires_at)
VALUES (?, ?, ?, ?, 'cimd', ?)
ON CONFLICT (client_id) DO UPDATE SET name = excluded.name, redirect_uris = excluded.redirect_uris, expires_at = excluded.expires_at
WHERE oauth_clients.kind = 'cimd';

-- name: GetOAuthClient :one
SELECT client_id, name, redirect_uris, ip, user_agent, origin, created_at, kind, secret_hash, expires_at FROM oauth_clients WHERE client_id = ?;

-- name: ListOAuthClients :many
SELECT client_id, name, redirect_uris, ip, user_agent, origin, created_at, kind, secret_hash, expires_at FROM oauth_clients ORDER BY created_at DESC;

-- name: DeleteOAuthClient :execresult
DELETE FROM oauth_clients WHERE client_id = ?;

-- name: DeleteAPITokensOfOAuthClient :exec
DELETE FROM api_tokens WHERE id IN (SELECT r.api_token_id FROM oauth_refresh_tokens r WHERE r.client_id = ?);

-- name: CountOAuthClients :one
SELECT COUNT(*) FROM oauth_clients;

-- name: DeleteIdleOAuthClients :execresult
DELETE FROM oauth_clients
WHERE oauth_clients.created_at < ?
  AND oauth_clients.kind <> 'manual'
  AND oauth_clients.client_id NOT IN (SELECT r.client_id FROM oauth_refresh_tokens r);

-- name: InsertOAuthRefresh :exec
INSERT INTO oauth_refresh_tokens (token_hash, client_id, user_id, api_token_id, expires_at, created_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetOAuthRefreshByHash :one
SELECT id, token_hash, client_id, user_id, api_token_id, expires_at, created_at
FROM oauth_refresh_tokens WHERE token_hash = ?;

-- name: RotateOAuthRefresh :execresult
UPDATE oauth_refresh_tokens SET token_hash = ?, api_token_id = ?, expires_at = ?
WHERE id = ? AND token_hash = ?;

-- name: DeleteOAuthRefresh :exec
DELETE FROM oauth_refresh_tokens WHERE id = ?;

-- name: DeleteExpiredOAuthRefresh :exec
DELETE FROM oauth_refresh_tokens WHERE expires_at < ?;

-- name: ListOAuthCIMDHosts :many
SELECT host FROM oauth_cimd_hosts ORDER BY host;

-- name: InsertOAuthCIMDHost :exec
INSERT INTO oauth_cimd_hosts (host, created_at) VALUES (?, ?) ON CONFLICT (host) DO NOTHING;

-- name: DeleteOAuthCIMDHost :execresult
DELETE FROM oauth_cimd_hosts WHERE host = ?;

-- name: CountOAuthCIMDHost :one
SELECT COUNT(*) FROM oauth_cimd_hosts WHERE host = ?;
