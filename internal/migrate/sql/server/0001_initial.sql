-- Server schema: users, authentication, SSO, instance settings and
-- everything else that does not belong to a space.


CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TEXT NOT NULL
) STRICT;
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE COLLATE NOCASE,
    password TEXT,
    -- Server role. A guest cannot create spaces or be a space admin; what a
    -- user may do inside a space is the space_members role.
    role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user', 'guest')),
    is_sso_only INTEGER NOT NULL DEFAULT 0 CHECK (is_sso_only IN (0, 1)),
    two_factor_secret TEXT,
    two_factor_enabled INTEGER NOT NULL DEFAULT 0 CHECK (two_factor_enabled IN (0, 1)),
    two_factor_confirmed INTEGER NOT NULL DEFAULT 0 CHECK (two_factor_confirmed IN (0, 1)),
    -- Deleted users are kept anonymised, so ids recorded in space databases
    -- (created_by) still resolve.
    deleted_at TEXT,
    created_at TEXT,
    updated_at TEXT
) STRICT;
CREATE TABLE IF NOT EXISTS auth_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    csrf TEXT NOT NULL,
    ip TEXT,
    user_agent TEXT,
    remember_me INTEGER NOT NULL DEFAULT 0 CHECK (remember_me IN (0, 1)),
    last_used_at TEXT NOT NULL,
    refreshed_at TEXT,
    idle_expires_at TEXT NOT NULL,
    absolute_expires_at TEXT NOT NULL,
    revoked_at TEXT,
    created_at TEXT,
    updated_at TEXT
) STRICT;
CREATE INDEX IF NOT EXISTS auth_sessions_user_revoked_idx ON auth_sessions (user_id, revoked_at);
CREATE INDEX IF NOT EXISTS auth_sessions_idle_idx ON auth_sessions (idle_expires_at);
CREATE TABLE IF NOT EXISTS password_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    consumed_at TEXT,
    created_at TEXT,
    updated_at TEXT
) STRICT;
CREATE INDEX IF NOT EXISTS password_tokens_user_consumed_idx ON password_tokens (user_id, consumed_at);
CREATE INDEX IF NOT EXISTS password_tokens_expires_idx ON password_tokens (expires_at);
CREATE TABLE IF NOT EXISTS two_factor_challenges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    consumed_at TEXT,
    created_at TEXT,
    updated_at TEXT
) STRICT;
CREATE INDEX IF NOT EXISTS two_factor_challenges_user_idx ON two_factor_challenges (user_id);
CREATE INDEX IF NOT EXISTS two_factor_challenges_expires_idx ON two_factor_challenges (expires_at);
CREATE TABLE IF NOT EXISTS two_factor_recovery_codes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    used_at TEXT,
    created_at TEXT
) STRICT;
CREATE INDEX IF NOT EXISTS two_factor_recovery_codes_user_idx ON two_factor_recovery_codes (user_id);
CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id TEXT NOT NULL UNIQUE,
    name TEXT,
    aaguid TEXT,
    record TEXT NOT NULL,
    counter INTEGER NOT NULL DEFAULT 0,
    last_used_at TEXT,
    created_at TEXT,
    updated_at TEXT
) STRICT;
CREATE INDEX IF NOT EXISTS webauthn_credentials_user_idx ON webauthn_credentials (user_id);
CREATE TABLE IF NOT EXISTS webauthn_challenges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    options TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    consumed_at TEXT,
    created_at TEXT,
    updated_at TEXT
) STRICT;
CREATE INDEX IF NOT EXISTS webauthn_challenges_user_idx ON webauthn_challenges (user_id);
CREATE INDEX IF NOT EXISTS webauthn_challenges_expires_idx ON webauthn_challenges (expires_at);
CREATE TABLE IF NOT EXISTS api_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    prefix TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT 'read' CHECK (scope IN ('read', 'read-write')),
    expires_at TEXT,
    last_used_at TEXT,
    created_at TEXT,
    updated_at TEXT
) STRICT;
CREATE INDEX IF NOT EXISTS api_tokens_user_idx ON api_tokens (user_id);
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT
) STRICT;
CREATE TABLE IF NOT EXISTS uploads (
    id TEXT PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    bucket TEXT NOT NULL,
    object_key TEXT NOT NULL,
    disk TEXT NOT NULL,
    path TEXT,
    original_name TEXT NOT NULL,
    mime_type TEXT,
    size INTEGER,
    part_size INTEGER,
    total_parts INTEGER,
    status TEXT NOT NULL DEFAULT 'pending',
    completed_at TEXT,
    expires_at TEXT,
    created_at TEXT,
    updated_at TEXT
) STRICT;
CREATE INDEX IF NOT EXISTS uploads_bucket_idx ON uploads (bucket);
CREATE INDEX IF NOT EXISTS uploads_status_expires_idx ON uploads (status, expires_at);
CREATE INDEX IF NOT EXISTS uploads_user_status_idx ON uploads (user_id, status);
CREATE TABLE IF NOT EXISTS identity_providers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    protocol TEXT NOT NULL,
    preset TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
    sort_order INTEGER NOT NULL DEFAULT 0,
    config TEXT,
    secrets TEXT,
    claim_mappings TEXT,
    role_mapping TEXT,
    default_role TEXT NOT NULL DEFAULT 'user',
    allow_jit INTEGER NOT NULL DEFAULT 1 CHECK (allow_jit IN (0, 1)),
    sync_role_on_login INTEGER NOT NULL DEFAULT 0 CHECK (sync_role_on_login IN (0, 1)),
    link_by_email INTEGER NOT NULL DEFAULT 1 CHECK (link_by_email IN (0, 1)),
    created_at TEXT,
    updated_at TEXT
) STRICT;
CREATE INDEX IF NOT EXISTS identity_providers_enabled_idx ON identity_providers (enabled, sort_order);
CREATE TABLE IF NOT EXISTS user_identities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    identity_provider_id INTEGER NOT NULL REFERENCES identity_providers(id) ON DELETE CASCADE,
    subject TEXT NOT NULL,
    last_login_at TEXT,
    claims TEXT,
    created_at TEXT,
    updated_at TEXT,
    UNIQUE (identity_provider_id, subject)
) STRICT;
CREATE INDEX IF NOT EXISTS user_identities_user_idx ON user_identities (user_id);
CREATE TABLE IF NOT EXISTS sso_login_states (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    state TEXT NOT NULL UNIQUE,
    identity_provider_id INTEGER NOT NULL REFERENCES identity_providers(id) ON DELETE CASCADE,
    nonce TEXT,
    code_verifier TEXT,
    saml_request_id TEXT,
    redirect_after TEXT,
    expires_at TEXT NOT NULL,
    created_at TEXT,
    updated_at TEXT
) STRICT;
CREATE INDEX IF NOT EXISTS sso_login_states_provider_idx ON sso_login_states (identity_provider_id);
CREATE INDEX IF NOT EXISTS sso_login_states_expires_idx ON sso_login_states (expires_at);
CREATE INDEX IF NOT EXISTS sso_login_states_saml_idx ON sso_login_states (saml_request_id);
CREATE TABLE IF NOT EXISTS sso_login_tickets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket TEXT NOT NULL UNIQUE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    requires_2fa INTEGER NOT NULL DEFAULT 0 CHECK (requires_2fa IN (0, 1)),
    two_factor_token TEXT,
    expires_at TEXT NOT NULL,
    consumed_at TEXT,
    created_at TEXT,
    updated_at TEXT
) STRICT;
CREATE INDEX IF NOT EXISTS sso_login_tickets_user_idx ON sso_login_tickets (user_id);
CREATE INDEX IF NOT EXISTS sso_login_tickets_expires_idx ON sso_login_tickets (expires_at);

-- Spaces: their databases live elsewhere (one per space); this is the
-- registry and who may enter them.
CREATE TABLE IF NOT EXISTS spaces (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    -- Size limit in bytes set by a server admin; NULL uses space_quota_mb.
    quota_bytes INTEGER CHECK (quota_bytes > 0),
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at TEXT,
    updated_at TEXT
) STRICT;
CREATE INDEX IF NOT EXISTS spaces_created_by_idx ON spaces (created_by);

CREATE TABLE IF NOT EXISTS space_members (
    space_id INTEGER NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('admin', 'editor', 'viewer')),
    created_at TEXT,
    PRIMARY KEY (space_id, user_id)
) STRICT;
CREATE INDEX IF NOT EXISTS space_members_user_idx ON space_members (user_id);

CREATE TABLE IF NOT EXISTS space_invitations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    space_id INTEGER NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    -- When set, only a user with this email may accept.
    email TEXT COLLATE NOCASE,
    role TEXT NOT NULL CHECK (role IN ('admin', 'editor', 'viewer')),
    token_hash TEXT NOT NULL UNIQUE,
    invited_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    -- Who registers through an invitation from a server admin becomes a user,
    -- through anyone else's a guest.
    invited_by_server_admin INTEGER NOT NULL DEFAULT 0 CHECK (invited_by_server_admin IN (0, 1)),
    expires_at TEXT NOT NULL,
    accepted_at TEXT,
    accepted_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at TEXT
) STRICT;
CREATE INDEX IF NOT EXISTS space_invitations_space_idx ON space_invitations (space_id);
CREATE INDEX IF NOT EXISTS space_invitations_invited_by_idx ON space_invitations (invited_by);
CREATE INDEX IF NOT EXISTS space_invitations_accepted_by_idx ON space_invitations (accepted_by);

-- Actions of server admins that reach into spaces (assigning an admin,
-- deleting or restoring a space, quotas, keys). The admins of the space see
-- the rows about it. space_id has no foreign key: the row outlives the space.
CREATE TABLE IF NOT EXISTS admin_audit (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    actor_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    space_id INTEGER,
    target_user_id INTEGER,
    details TEXT,
    created_at TEXT NOT NULL
) STRICT;
CREATE INDEX IF NOT EXISTS admin_audit_space_idx ON admin_audit (space_id, created_at);
CREATE INDEX IF NOT EXISTS admin_audit_actor_idx ON admin_audit (actor_id);

-- Linked spaces may transfer to each other. Only who administers both may
-- link them; an admin of either may unlink. space_a_id < space_b_id.
CREATE TABLE IF NOT EXISTS space_links (
    space_a_id INTEGER NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    space_b_id INTEGER NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at TEXT,
    PRIMARY KEY (space_a_id, space_b_id),
    CHECK (space_a_id < space_b_id)
) STRICT;
CREATE INDEX IF NOT EXISTS space_links_b_idx ON space_links (space_b_id);
CREATE INDEX IF NOT EXISTS space_links_created_by_idx ON space_links (created_by);

-- Public keys whose signatures this server accepts besides its own: the
-- keys it used before a rotation and keys of servers spaces moved from.
CREATE TABLE IF NOT EXISTS trusted_keys (
    kid TEXT PRIMARY KEY,
    public_key TEXT NOT NULL,
    name TEXT NOT NULL,
    added_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at TEXT
) STRICT;
CREATE INDEX IF NOT EXISTS trusted_keys_added_by_idx ON trusted_keys (added_by);
