-- Laravel-era database used by tests. The DDL mirrors a real Laravel backup
-- (declared types numeric/date/datetime, columns added by later migrations,
-- check and foreign key constraints). The data mirrors what Laravel stores:
-- integer-valued money in numeric columns reads back as INTEGER, fractional
-- money as REAL, and dates carry a "00:00:00" time part.
--
-- currencies:   1 USD (2 dec), 2 EUR (base, 2 dec), 3 JPY (0 dec), 4 BTC (8 dec)
-- accounts:     1 Cash USD, 2 Yen JPY, 3 Wallet BTC, 4 Mortgage (USD debt)
-- transactions: 1 expense, 2 expense, 3 income, 4 transfer, 5 pending
--
-- The row recording the last supported migration is added by CreateLaravelShape.

CREATE TABLE "migrations" ("id" integer primary key autoincrement not null, "migration" varchar not null, "batch" integer not null);
INSERT INTO migrations (migration, batch) VALUES ('2014_10_12_000000_create_users_table', 1);

CREATE TABLE "users" ("id" integer primary key autoincrement not null, "name" varchar not null, "email" varchar not null, "password" varchar, "created_at" datetime, "updated_at" datetime, "role" varchar not null default ('admin'), "two_factor_secret" text, "two_factor_enabled" tinyint(1) not null default ('0'), "two_factor_confirmed" tinyint(1) not null default ('0'), "is_sso_only" tinyint(1) not null default ('0'));
INSERT INTO users (id, name, email, password, role, created_at, updated_at)
VALUES (1, 'Ada', 'ada@example.com', '$2y$10$legacyhash', 'admin', '2026-01-02 14:39:00', '2026-01-02 14:39:00');

CREATE TABLE "currencies" ("id" integer primary key autoincrement not null, "code" varchar not null, "name" varchar not null, "symbol" varchar not null, "decimals" integer not null default '2', "created_at" datetime, "updated_at" datetime, "is_base" tinyint(1) not null default '0', "rate" numeric not null default '1');
INSERT INTO currencies (id, code, name, symbol, decimals, is_base, rate) VALUES (1, 'USD', 'US Dollar', '$', 2, 0, 0.88243363194593);
INSERT INTO currencies (id, code, name, symbol, decimals, is_base, rate) VALUES (2, 'EUR', 'Euro', '€', 2, 1, 1);
INSERT INTO currencies (id, code, name, symbol, decimals, is_base, rate) VALUES (3, 'JPY', 'Yen', '¥', 0, 0, 0.0067);
INSERT INTO currencies (id, code, name, symbol, decimals, is_base, rate) VALUES (4, 'BTC', 'Bitcoin', '₿', 8, 0, 50000);

CREATE TABLE "categories" ("id" integer primary key autoincrement not null, "name" varchar not null, "type" varchar check ("type" in ('income', 'expense')) not null, "icon" varchar, "color" varchar, "created_at" datetime, "updated_at" datetime);
INSERT INTO categories (id, name, type) VALUES (1, 'Groceries', 'expense');

CREATE TABLE "accounts" (
    "id" integer primary key autoincrement not null,
    "name" varchar not null,
    "type" varchar check ("type" in ('bank', 'crypto', 'cash', 'debt')) not null,
    "debt_type" varchar check ("debt_type" in ('i_owe', 'owed_to_me') OR "debt_type" IS NULL),
    "currency_id" integer not null,
    "initial_balance" numeric not null default '0',
    "target_amount" numeric,
    "due_date" date,
    "is_paid_off" tinyint(1) not null default '0',
    "counterparty" varchar,
    "debt_description" text,
    "is_active" tinyint(1) not null default '1',
    "created_at" datetime,
    "updated_at" datetime, "sort_order" integer not null default '0',
    foreign key("currency_id") references "currencies"("id")
);
INSERT INTO accounts (id, name, type, currency_id, initial_balance, is_active) VALUES (1, 'Cash', 'cash', 1, 100, 1);
INSERT INTO accounts (id, name, type, currency_id, initial_balance, is_active) VALUES (2, 'Yen', 'cash', 3, 3000, 1);
INSERT INTO accounts (id, name, type, currency_id, initial_balance, is_active) VALUES (3, 'Wallet', 'crypto', 4, 0.5, 1);
INSERT INTO accounts (id, name, type, debt_type, currency_id, initial_balance, target_amount, due_date, counterparty, is_active)
VALUES (4, 'Mortgage', 'debt', 'i_owe', 1, 0, 1200.5, '2027-08-15 00:00:00', 'Bank', 1);

CREATE TABLE "recurring_transactions" ("id" integer primary key autoincrement not null, "type" varchar check ("type" in ('income', 'expense', 'transfer')) not null, "account_id" integer not null, "to_account_id" integer, "category_id" integer, "amount" numeric not null, "to_amount" numeric, "description" varchar, "frequency" varchar check ("frequency" in ('daily', 'weekly', 'monthly', 'yearly')) not null, "interval" integer not null default '1', "day_of_week" integer, "day_of_month" integer, "start_date" date not null, "end_date" date, "next_run_date" date not null, "last_run_date" date, "is_active" tinyint(1) not null default '1', "created_at" datetime, "updated_at" datetime, foreign key("account_id") references "accounts"("id") on delete cascade, foreign key("to_account_id") references "accounts"("id") on delete set null, foreign key("category_id") references "categories"("id") on delete set null);
INSERT INTO recurring_transactions (id, type, account_id, category_id, amount, description, frequency, day_of_month, start_date, end_date, next_run_date, last_run_date)
VALUES (1, 'expense', 1, 1, 15, 'Gym', 'monthly', 15, '2026-08-15 00:00:00', '2027-06-15 00:00:00', '2026-10-15 00:00:00', '2026-09-15 00:00:00');

CREATE TABLE "transactions" ("id" integer primary key autoincrement not null, "type" varchar not null, "account_id" integer not null, "to_account_id" integer, "category_id" integer, "amount" numeric not null, "to_amount" numeric, "exchange_rate" numeric, "description" varchar, "dedup_hash" varchar, "date" date, "created_at" datetime, "updated_at" datetime, "status" varchar not null default ('confirmed'), "recurring_transaction_id" integer, foreign key("category_id") references categories("id") on delete no action on update no action, foreign key("to_account_id") references accounts("id") on delete no action on update no action, foreign key("account_id") references accounts("id") on delete no action on update no action, foreign key("recurring_transaction_id") references recurring_transactions("id") on delete set null on update no action);
INSERT INTO transactions (id, type, account_id, category_id, amount, description, date, created_at, updated_at)
VALUES (1, 'expense', 1, 1, 12.5, 'Coffee', '2026-01-03 00:00:00', '2026-01-03 09:15:00', '2026-01-03 09:15:00');
INSERT INTO transactions (id, type, account_id, amount, description, date, created_at, updated_at)
VALUES (2, 'expense', 2, 1500, 'Ramen', '2026-01-04 00:00:00', '2026-01-04 12:00:00', '2026-01-04 12:00:00');
INSERT INTO transactions (id, type, account_id, amount, description, date, created_at, updated_at)
VALUES (3, 'income', 3, 0.00012345, 'Mining', '2026-01-05 00:00:00', '2026-01-05 08:00:00', '2026-01-05 08:00:00');
INSERT INTO transactions (id, type, account_id, to_account_id, amount, to_amount, exchange_rate, description, date, created_at, updated_at)
VALUES (4, 'transfer', 1, 2, 10.5, 1567, 149.2, 'Exchange', '2026-01-06 00:00:00', '2026-01-06 10:00:00', '2026-01-06 10:00:00');
INSERT INTO transactions (id, type, account_id, category_id, amount, description, date, status, recurring_transaction_id, created_at, updated_at)
VALUES (5, 'expense', 1, 1, 15, 'Gym', '2026-10-15 00:00:00', 'pending', 1, '2026-09-15 00:00:00', '2026-09-15 00:00:00');

CREATE TABLE "transaction_items" ("id" integer primary key autoincrement not null, "transaction_id" integer not null, "name" varchar not null, "quantity" numeric not null default '1', "price_per_unit" numeric not null, "total_price" numeric not null, "created_at" datetime, "updated_at" datetime, foreign key("transaction_id") references transactions("id") on delete cascade on update no action);
INSERT INTO transaction_items (id, transaction_id, name, quantity, price_per_unit, total_price) VALUES (1, 1, 'Beans', 5, 2.5, 12.5);
INSERT INTO transaction_items (id, transaction_id, name, quantity, price_per_unit, total_price) VALUES (2, 2, 'Noodles', 0.5, 3000, 1500);

CREATE TABLE "budgets" ("id" integer primary key autoincrement not null, "name" varchar not null, "amount" numeric not null, "period" varchar not null, "start_date" date, "end_date" date, "is_global" tinyint(1) not null default ('0'), "notify_at_percent" integer, "is_active" tinyint(1) not null default ('1'), "created_at" datetime, "updated_at" datetime, "currency_id" integer, foreign key("currency_id") references "currencies"("id") on delete set null);
INSERT INTO budgets (id, name, amount, period, currency_id) VALUES (1, 'Food', 750.5, 'monthly', 2);
INSERT INTO budgets (id, name, amount, period, start_date, end_date, currency_id) VALUES (2, 'Trip', 300, 'monthly', '2026-08-01 00:00:00', '2026-08-31 00:00:00', NULL);

CREATE TABLE jobs (id INTEGER PRIMARY KEY, queue TEXT);
