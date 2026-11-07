-- Space schema: the finances of one space. Each space has its own database,
-- so no table carries a space id. Every amount is an exact integer in minor
-- units of its own currency. v1.0 files (space/0001_initial) are converted into
-- a file with this base by internal/migrate/upgrade/v1, never migrated in place; see
-- docs/plans/transactions-erd.md.

CREATE TABLE IF NOT EXISTS currencies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT NOT NULL UNIQUE COLLATE NOCASE,
    name TEXT NOT NULL,
    symbol TEXT NOT NULL,
    decimals INTEGER NOT NULL DEFAULT 2 CHECK (decimals BETWEEN 0 AND 12),
    is_base INTEGER NOT NULL DEFAULT 0 CHECK (is_base IN (0, 1)),
    rate TEXT NOT NULL DEFAULT '1',
    created_at TEXT,
    updated_at TEXT
) STRICT;
CREATE UNIQUE INDEX IF NOT EXISTS currencies_single_base_unique ON currencies (is_base) WHERE is_base = 1;

-- A debt is an account of type debt (its data is in debts); a space account
-- stands for all other spaces in one currency. Both are service accounts.
CREATE TABLE IF NOT EXISTS accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('bank', 'crypto', 'cash', 'credit', 'debt', 'space')),
    is_service INTEGER GENERATED ALWAYS AS (type IN ('debt', 'space')) VIRTUAL,
    currency_id INTEGER NOT NULL REFERENCES currencies(id),
    initial_balance INTEGER NOT NULL DEFAULT 0,
    is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT,
    updated_at TEXT
) STRICT;
CREATE INDEX IF NOT EXISTS accounts_currency_idx ON accounts (currency_id);
CREATE UNIQUE INDEX IF NOT EXISTS accounts_space_currency_unique ON accounts (currency_id) WHERE type = 'space';

-- A credit card is an ordinary account of type credit; this is how its billing
-- period runs. The app only shows its state (limit, statement, due date): it
-- computes no interest or fees. fixed_day closes a period on statement_day of
-- every month; first_expense (a period that starts when the balance goes below
-- zero and lasts period_days) is kept for later, the service refuses it.
CREATE TABLE IF NOT EXISTS credit_cards (
    account_id INTEGER PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    credit_limit INTEGER NOT NULL CHECK (credit_limit > 0),
    cycle TEXT NOT NULL CHECK (cycle IN ('fixed_day', 'first_expense')),
    statement_day INTEGER CHECK (statement_day BETWEEN 1 AND 31),
    period_days INTEGER CHECK (period_days > 0),
    grace_days INTEGER NOT NULL CHECK (grace_days >= 0),
    CHECK (
        (cycle = 'fixed_day' AND statement_day IS NOT NULL AND period_days IS NULL)
        OR (cycle = 'first_expense' AND period_days IS NOT NULL AND statement_day IS NULL)
    )
) STRICT;

CREATE TABLE IF NOT EXISTS debts (
    account_id INTEGER PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    direction TEXT NOT NULL CHECK (direction IN ('i_owe', 'owed_to_me')),
    principal INTEGER NOT NULL CHECK (principal > 0),
    due_date TEXT,
    counterparty TEXT,
    note TEXT,
    closed_at TEXT
) STRICT;

CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE,
    type TEXT NOT NULL CHECK (type IN ('income', 'expense')),
    icon TEXT,
    color TEXT,
    is_default INTEGER NOT NULL DEFAULT 0 CHECK (is_default IN (0, 1)),
    created_at TEXT,
    updated_at TEXT
) STRICT;
CREATE UNIQUE INDEX IF NOT EXISTS categories_type_name_unique ON categories (type, name COLLATE NOCASE);

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE,
    created_at TEXT,
    updated_at TEXT
) STRICT;

-- A transaction is facts only: when, what kind, and who made it. Money that
-- moved is in payments, what it was spent on or earned from is in
-- transaction_lines. No constraint ties the two; internal/ledger compares them.
CREATE TABLE IF NOT EXISTS transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    -- transfer_out / transfer_in are this space's side of a transfer with a
    -- linked space (see space_transfers); they are written only from there.
    type TEXT NOT NULL CHECK (type IN ('income', 'expense', 'transfer', 'transfer_out', 'transfer_in', 'debt_lend', 'debt_borrow', 'debt_payment', 'debt_collection')),
    description TEXT,
    date TEXT,
    status TEXT NOT NULL DEFAULT 'confirmed' CHECK (status IN ('pending', 'confirmed', 'skipped')),
    -- Manual order within a day. NULL sorts by id; set only by hand.
    day_order INTEGER,
    -- Set by the importer; not unique, the importer alone prevents duplicates.
    external_id TEXT,
    -- A server user; no foreign key across databases.
    created_by INTEGER,
    created_at TEXT,
    updated_at TEXT
) STRICT;
-- The list is ordered by date, then manual day order (id when unset), then id;
-- undated rows come last in either direction. These match txOrderBy in
-- internal/db/filter, so a page is read in index order and stops at LIMIT
-- instead of sorting every matching row. The descending one also serves date ranges.
CREATE INDEX IF NOT EXISTS transactions_order_desc_idx ON transactions (date DESC, (COALESCE(day_order, id)) DESC, id DESC);
CREATE INDEX IF NOT EXISTS transactions_order_asc_idx ON transactions ((date IS NULL), date, (COALESCE(day_order, id)), id);
CREATE INDEX IF NOT EXISTS transactions_external_id_idx ON transactions (external_id) WHERE external_id IS NOT NULL;

-- Money that moved, signed, in the account's currency. One per account in a
-- transaction. 0 is allowed only because v1.0 rows have it; the ledger rejects
-- it on new input. A payment has no type: its role follows from its sign and
-- the transaction type.
CREATE TABLE IF NOT EXISTS payments (
    transaction_id INTEGER NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    amount INTEGER NOT NULL,
    -- Set only on a payment on a space account: the cross-space movement.
    space_transfer_uuid TEXT REFERENCES space_transfers(uuid) ON DELETE RESTRICT,
    PRIMARY KEY (transaction_id, account_id)
) STRICT, WITHOUT ROWID;
CREATE INDEX IF NOT EXISTS payments_account_idx ON payments (account_id, transaction_id, amount);
CREATE UNIQUE INDEX IF NOT EXISTS payments_space_transfer_unique ON payments (space_transfer_uuid) WHERE space_transfer_uuid IS NOT NULL;

-- What it was spent on or earned from: an amount for a category, not a
-- product. The sign comes from the transaction type. A NULL category is an
-- uncategorised line (v1.0 rows allow it).
CREATE TABLE IF NOT EXISTS transaction_lines (
    transaction_id INTEGER NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    seq INTEGER NOT NULL,
    category_id INTEGER REFERENCES categories(id) ON DELETE RESTRICT,
    currency_id INTEGER NOT NULL REFERENCES currencies(id),
    amount INTEGER NOT NULL CHECK (amount > 0),
    note TEXT,
    PRIMARY KEY (transaction_id, seq)
) STRICT, WITHOUT ROWID;
CREATE INDEX IF NOT EXISTS transaction_lines_category_idx ON transaction_lines (category_id, currency_id, amount);
CREATE INDEX IF NOT EXISTS transaction_lines_currency_idx ON transaction_lines (currency_id);

CREATE TABLE IF NOT EXISTS transaction_tag (
    transaction_id INTEGER NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (transaction_id, tag_id)
) STRICT;
CREATE INDEX IF NOT EXISTS transaction_tag_tag_idx ON transaction_tag (tag_id);

-- template is the transaction input of the API (what to create, tags included);
-- schedule is when. Both are validated by Go; the database checks only that
-- they are JSON. The next due date is derived from recurring_instances.
CREATE TABLE IF NOT EXISTS recurring_transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    template TEXT NOT NULL CHECK (json_valid(template)),
    schedule TEXT NOT NULL CHECK (json_valid(schedule)),
    is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    -- Why the rule was stopped (invalid template or schedule).
    error TEXT,
    created_at TEXT,
    updated_at TEXT
) STRICT;

-- One row per handled due date. The transaction is NULL once it is deleted:
-- the due date stays handled and is not generated again.
CREATE TABLE IF NOT EXISTS recurring_instances (
    recurring_transaction_id INTEGER NOT NULL REFERENCES recurring_transactions(id) ON DELETE CASCADE,
    due_date TEXT NOT NULL,
    transaction_id INTEGER UNIQUE REFERENCES transactions(id) ON DELETE SET NULL,
    PRIMARY KEY (recurring_transaction_id, due_date)
) STRICT, WITHOUT ROWID;

-- A transfer with a linked space. Both spaces hold the whole record, so each
-- space's backup has its transfers; they are kept in step by version (a
-- deletion is a tombstone with its own version). payload is the exact bytes
-- that were signed, never re-encoded; source, destination and version are read
-- from it (or from remote while the row is in review). status is local
-- bookkeeping: not signed, not synced.
CREATE TABLE IF NOT EXISTS space_transfers (
    uuid TEXT PRIMARY KEY,
    -- NULL only in review when this space lacks the record.
    payload TEXT CHECK (payload IS NULL OR json_valid(payload)),
    -- 1: signed by the server key (v1.0, until re-signed); 2: by both space keys.
    sig_v INTEGER NOT NULL DEFAULT 2 CHECK (sig_v IN (1, 2)),
    source_kid TEXT,
    source_sig TEXT,
    destination_kid TEXT,
    destination_sig TEXT,
    source TEXT GENERATED ALWAYS AS (COALESCE(payload, remote) ->> '$.from.space') VIRTUAL,
    destination TEXT GENERATED ALWAYS AS (COALESCE(payload, remote) ->> '$.to.space') VIRTUAL,
    version INTEGER GENERATED ALWAYS AS (COALESCE(payload, remote) ->> '$.version') VIRTUAL,
    status TEXT NOT NULL DEFAULT 'confirmed' CHECK (status IN ('confirmed', 'in_review')),
    -- The other space's version while in review, as JSON. NULL there means the
    -- other space lacks the record.
    remote TEXT CHECK (remote IS NULL OR json_valid(remote)),
    updated_at TEXT,
    CHECK (status = 'in_review' OR (payload IS NOT NULL AND remote IS NULL)),
    CHECK (payload IS NOT NULL OR remote IS NOT NULL)
) STRICT;
CREATE INDEX IF NOT EXISTS space_transfers_source_idx ON space_transfers (source);
CREATE INDEX IF NOT EXISTS space_transfers_destination_idx ON space_transfers (destination);

-- Ed25519 keys of this space. The seed travels with the space's backup, so a
-- transfer stays valid when both of its spaces move to another server. A
-- retired key still verifies and is never deleted while a transfer uses it.
CREATE TABLE IF NOT EXISTS space_keys (
    kid TEXT PRIMARY KEY,
    seed TEXT NOT NULL,
    created_at TEXT NOT NULL,
    retired_at TEXT
) STRICT;

CREATE TABLE IF NOT EXISTS budgets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    amount INTEGER NOT NULL CHECK (amount >= 0),
    currency_id INTEGER NOT NULL REFERENCES currencies(id),
    period TEXT NOT NULL CHECK (period IN ('weekly', 'monthly', 'yearly', 'one_time')),
    start_date TEXT,
    end_date TEXT,
    is_global INTEGER NOT NULL DEFAULT 0 CHECK (is_global IN (0, 1)),
    notify_at_percent INTEGER,
    is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    created_at TEXT,
    updated_at TEXT
) STRICT;
CREATE INDEX IF NOT EXISTS budgets_currency_idx ON budgets (currency_id);
CREATE TABLE IF NOT EXISTS budget_category (
    budget_id INTEGER NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (budget_id, category_id)
) STRICT;
CREATE INDEX IF NOT EXISTS budget_category_category_idx ON budget_category (category_id);
CREATE TABLE IF NOT EXISTS budget_tag (
    budget_id INTEGER NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (budget_id, tag_id)
) STRICT;
CREATE INDEX IF NOT EXISTS budget_tag_tag_idx ON budget_tag (tag_id);

CREATE TABLE IF NOT EXISTS automation_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    trigger_type TEXT NOT NULL CHECK (trigger_type IN ('on_transaction_create', 'on_transaction_update')),
    priority INTEGER NOT NULL DEFAULT 50,
    conditions TEXT NOT NULL,
    actions TEXT NOT NULL,
    is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    stop_processing INTEGER NOT NULL DEFAULT 0 CHECK (stop_processing IN (0, 1)),
    runs_count INTEGER NOT NULL DEFAULT 0,
    last_run_at TEXT,
    created_at TEXT,
    updated_at TEXT
) STRICT;
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
) STRICT;
CREATE INDEX IF NOT EXISTS automation_rule_logs_rule_idx ON automation_rule_logs (rule_id, created_at);

CREATE TABLE IF NOT EXISTS transaction_imports (
    id TEXT PRIMARY KEY,
    user_id INTEGER, -- a server user; no foreign key across databases
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
) STRICT;
CREATE INDEX IF NOT EXISTS transaction_imports_status_idx ON transaction_imports (status);
CREATE INDEX IF NOT EXISTS transaction_imports_user_status_idx ON transaction_imports (user_id, status);

-- Settings of this space (key/value), e.g. auto_update_currencies. Instance
-- settings live in the server database.
CREATE TABLE IF NOT EXISTS space_settings (
    key TEXT PRIMARY KEY,
    value TEXT
) STRICT;
