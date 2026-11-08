-- OAuth client kinds: dcr (self-registered), cimd (fetched from a Client ID
-- Metadata Document URL), manual (created by an admin, optional secret).
ALTER TABLE oauth_clients ADD COLUMN kind TEXT NOT NULL DEFAULT 'dcr';
-- Hash of the client secret; empty for public clients.
ALTER TABLE oauth_clients ADD COLUMN secret_hash TEXT NOT NULL DEFAULT '';
-- When a cimd client's cached document must be fetched again.
ALTER TABLE oauth_clients ADD COLUMN expires_at TEXT;
-- Hosts allowed to serve CIMD documents. Empty means CIMD is off.
CREATE TABLE IF NOT EXISTS oauth_cimd_hosts (
    host TEXT PRIMARY KEY,
    created_at TEXT
) STRICT;
