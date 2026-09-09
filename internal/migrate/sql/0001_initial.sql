-- Savvy domain schema (Go). Column names match the Laravel-era SQLite so
-- legacy import can copy rows with minimal remapping.

CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password TEXT,
    role TEXT NOT NULL DEFAULT 'admin',
    is_sso_only INTEGER NOT NULL DEFAULT 0,
    two_factor_secret TEXT,
    two_factor_enabled INTEGER NOT NULL DEFAULT 0,
    two_factor_confirmed INTEGER NOT NULL DEFAULT 0,
    created_at TEXT,
    updated_at TEXT
);

CREATE TABLE IF NOT EXISTS auth_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    csrf TEXT NOT NULL,
    ip TEXT,
    user_agent TEXT,
    remember_me INTEGER NOT NULL DEFAULT 0,
    last_used_at TEXT NOT NULL,
    refreshed_at TEXT,
    idle_expires_at TEXT NOT NULL,
    absolute_expires_at TEXT NOT NULL,
    revoked_at TEXT,
    created_at TEXT,
    updated_at TEXT
);
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
);
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
);
CREATE INDEX IF NOT EXISTS two_factor_challenges_expires_idx ON two_factor_challenges (expires_at);

CREATE TABLE IF NOT EXISTS two_factor_recovery_codes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    used_at TEXT,
    created_at TEXT
);

CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id TEXT NOT NULL UNIQUE,
    name TEXT,
    aaguid TEXT,
    record TEXT NOT NULL,
    transports TEXT,
    counter INTEGER NOT NULL DEFAULT 0,
    last_used_at TEXT,
    created_at TEXT,
    updated_at TEXT
);
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
);
CREATE INDEX IF NOT EXISTS webauthn_challenges_expires_idx ON webauthn_challenges (expires_at);

CREATE TABLE IF NOT EXISTS currencies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    symbol TEXT NOT NULL,
    decimals INTEGER NOT NULL DEFAULT 2,
    is_base INTEGER NOT NULL DEFAULT 0,
    rate REAL NOT NULL DEFAULT 1,
    created_at TEXT,
    updated_at TEXT
);

CREATE TABLE IF NOT EXISTS accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    debt_type TEXT,
    currency_id INTEGER NOT NULL REFERENCES currencies(id),
    initial_balance REAL NOT NULL DEFAULT 0,
    target_amount REAL,
    due_date TEXT,
    is_paid_off INTEGER NOT NULL DEFAULT 0,
    counterparty TEXT,
    debt_description TEXT,
    is_active INTEGER NOT NULL DEFAULT 1,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT,
    updated_at TEXT
);
CREATE INDEX IF NOT EXISTS accounts_type_debt_idx ON accounts (type, debt_type);
CREATE INDEX IF NOT EXISTS accounts_paid_off_idx ON accounts (is_paid_off);

CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    icon TEXT,
    color TEXT,
    created_at TEXT,
    updated_at TEXT
);

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    created_at TEXT,
    updated_at TEXT
);

CREATE TABLE IF NOT EXISTS recurring_transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL,
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    to_account_id INTEGER REFERENCES accounts(id) ON DELETE SET NULL,
    category_id INTEGER REFERENCES categories(id) ON DELETE SET NULL,
    amount REAL NOT NULL,
    to_amount REAL,
    description TEXT,
    frequency TEXT NOT NULL,
    interval INTEGER NOT NULL DEFAULT 1,
    day_of_week INTEGER,
    day_of_month INTEGER,
    start_date TEXT NOT NULL,
    end_date TEXT,
    next_run_date TEXT NOT NULL,
    last_run_date TEXT,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT,
    updated_at TEXT
);
CREATE INDEX IF NOT EXISTS recurring_next_run_idx ON recurring_transactions (next_run_date);

CREATE TABLE IF NOT EXISTS recurring_transaction_tag (
    recurring_transaction_id INTEGER NOT NULL REFERENCES recurring_transactions(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (recurring_transaction_id, tag_id)
);

CREATE TABLE IF NOT EXISTS transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL,
    account_id INTEGER NOT NULL REFERENCES accounts(id),
    to_account_id INTEGER REFERENCES accounts(id),
    category_id INTEGER REFERENCES categories(id),
    amount REAL NOT NULL,
    to_amount REAL,
    exchange_rate REAL,
    description TEXT,
    dedup_hash TEXT,
    date TEXT,
    status TEXT NOT NULL DEFAULT 'confirmed',
    recurring_transaction_id INTEGER REFERENCES recurring_transactions(id) ON DELETE SET NULL,
    created_at TEXT,
    updated_at TEXT
);
CREATE INDEX IF NOT EXISTS transactions_date_idx ON transactions (date);
CREATE INDEX IF NOT EXISTS transactions_type_idx ON transactions (type);
CREATE UNIQUE INDEX IF NOT EXISTS transactions_account_dedup_unique ON transactions (account_id, dedup_hash);
CREATE UNIQUE INDEX IF NOT EXISTS transactions_recurring_pending_unique
    ON transactions (recurring_transaction_id)
    WHERE status = 'pending' AND recurring_transaction_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS transactions_account_type_date_amount_idx ON transactions (account_id, type, date, amount);
CREATE INDEX IF NOT EXISTS transactions_to_account_date_amount_idx ON transactions (to_account_id, date, to_amount);
CREATE INDEX IF NOT EXISTS transactions_type_date_amount_idx ON transactions (type, date, amount);
CREATE INDEX IF NOT EXISTS transactions_account_date_idx ON transactions (account_id, date);

CREATE TABLE IF NOT EXISTS transaction_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    transaction_id INTEGER NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    quantity REAL NOT NULL DEFAULT 1,
    unit TEXT,
    price_per_unit REAL NOT NULL,
    total_price REAL NOT NULL,
    created_at TEXT,
    updated_at TEXT
);

CREATE TABLE IF NOT EXISTS transaction_tag (
    transaction_id INTEGER NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (transaction_id, tag_id)
);

CREATE TABLE IF NOT EXISTS budgets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    amount REAL NOT NULL,
    currency_id INTEGER REFERENCES currencies(id) ON DELETE SET NULL,
    period TEXT NOT NULL,
    start_date TEXT,
    end_date TEXT,
    is_global INTEGER NOT NULL DEFAULT 0,
    notify_at_percent INTEGER,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT,
    updated_at TEXT
);

CREATE TABLE IF NOT EXISTS budget_category (
    budget_id INTEGER NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (budget_id, category_id)
);

CREATE TABLE IF NOT EXISTS budget_tag (
    budget_id INTEGER NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (budget_id, tag_id)
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT
);

CREATE TABLE IF NOT EXISTS automation_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    trigger_type TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 50,
    conditions TEXT NOT NULL,
    actions TEXT NOT NULL,
    is_active INTEGER NOT NULL DEFAULT 1,
    stop_processing INTEGER NOT NULL DEFAULT 0,
    runs_count INTEGER NOT NULL DEFAULT 0,
    last_run_at TEXT,
    created_at TEXT,
    updated_at TEXT
);
CREATE INDEX IF NOT EXISTS automation_rules_trigger_idx ON automation_rules (trigger_type, is_active, priority);

CREATE TABLE IF NOT EXISTS automation_rule_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id INTEGER NOT NULL REFERENCES automation_rules(id) ON DELETE CASCADE,
    trigger_entity_type TEXT,
    trigger_entity_id INTEGER,
    actions_executed TEXT,
    status TEXT NOT NULL,
    error_message TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS automation_rule_logs_rule_idx ON automation_rule_logs (rule_id, created_at);

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
);
CREATE INDEX IF NOT EXISTS uploads_bucket_idx ON uploads (bucket);
CREATE INDEX IF NOT EXISTS uploads_status_expires_idx ON uploads (status, expires_at);
CREATE INDEX IF NOT EXISTS uploads_user_status_idx ON uploads (user_id, status);

CREATE TABLE IF NOT EXISTS transaction_imports (
    id TEXT PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    upload_id TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    mapping TEXT,
    options TEXT,
    total_rows INTEGER,
    processed_rows INTEGER NOT NULL DEFAULT 0,
    created_count INTEGER NOT NULL DEFAULT 0,
    skipped_count INTEGER NOT NULL DEFAULT 0,
    error_count INTEGER NOT NULL DEFAULT 0,
    errors TEXT,
    meta TEXT,
    message TEXT,
    created_at TEXT,
    updated_at TEXT
);
CREATE INDEX IF NOT EXISTS transaction_imports_status_idx ON transaction_imports (status);
CREATE INDEX IF NOT EXISTS transaction_imports_user_status_idx ON transaction_imports (user_id, status);

CREATE TABLE IF NOT EXISTS backups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    filename TEXT NOT NULL,
    size INTEGER NOT NULL,
    note TEXT,
    app_version TEXT,
    schema_migrations TEXT,
    created_at TEXT,
    updated_at TEXT
);

CREATE TABLE IF NOT EXISTS identity_providers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    protocol TEXT NOT NULL,
    preset TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    config TEXT,
    secrets TEXT,
    claim_mappings TEXT,
    role_mapping TEXT,
    default_role TEXT NOT NULL DEFAULT 'read-only',
    allow_jit INTEGER NOT NULL DEFAULT 1,
    sync_role_on_login INTEGER NOT NULL DEFAULT 0,
    link_by_email INTEGER NOT NULL DEFAULT 1,
    created_at TEXT,
    updated_at TEXT
);
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
);
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
);
CREATE INDEX IF NOT EXISTS sso_login_states_expires_idx ON sso_login_states (expires_at);
CREATE INDEX IF NOT EXISTS sso_login_states_saml_idx ON sso_login_states (saml_request_id);

CREATE TABLE IF NOT EXISTS sso_login_tickets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket TEXT NOT NULL UNIQUE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    requires_2fa INTEGER NOT NULL DEFAULT 0,
    two_factor_token TEXT,
    expires_at TEXT NOT NULL,
    consumed_at TEXT,
    created_at TEXT,
    updated_at TEXT
);
CREATE INDEX IF NOT EXISTS sso_login_tickets_expires_idx ON sso_login_tickets (expires_at);
