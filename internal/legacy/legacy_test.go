package legacy

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db"
	"savvy-go/internal/domain"
	"savvy-go/internal/migrate"
)

func TestCopyFromLaravelFixture(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	src, err := db.Open(filepath.Join(dir, "laravel.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = src.Close() })
	createLaravelShape(t, src)

	dest, err := db.Open(filepath.Join(dir, "go.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dest.Close() })
	if err := migrate.Up(ctx, dest); err != nil {
		t.Fatal(err)
	}

	if !IsLaravel(ctx, src) {
		t.Fatal("expected laravel detection")
	}
	if IsLaravel(ctx, dest) {
		t.Fatal("fresh go db should not look like laravel")
	}

	if err := Copy(ctx, dest, src); err != nil {
		t.Fatal(err)
	}
	assertCopied(t, dest)
	if !AlreadyImported(ctx, dest) {
		t.Fatal("expected import stamp")
	}

	// Idempotent: second copy does not duplicate.
	if err := Copy(ctx, dest, src); err != nil {
		t.Fatal(err)
	}
	assertCopied(t, dest)
}

func TestUpgradeInPlace(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "database.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	createLaravelShape(t, sqlDB)
	if err := EnsureColumns(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	if err := migrate.Up(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	if err := UpgradeInPlace(ctx, sqlDB, ""); err != nil {
		t.Fatal(err)
	}
	if err := UpgradeInPlace(ctx, sqlDB, ""); err != nil {
		t.Fatal(err)
	}
	if !AlreadyImported(ctx, sqlDB) {
		t.Fatal("expected stamp")
	}
	if tableExists(ctx, sqlDB, "migrations") {
		t.Fatal("laravel migrations table should be dropped")
	}
	var email string
	assertMinorUnits(t, sqlDB)
	if err := sqlDB.QueryRow(`SELECT email FROM users WHERE id = 1`).Scan(&email); err != nil {
		t.Fatal(err)
	}
	if email != "ada@example.com" {
		t.Fatalf("email %q", email)
	}
}

func createLaravelShape(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE migrations (id INTEGER PRIMARY KEY AUTOINCREMENT, migration TEXT NOT NULL, batch INTEGER NOT NULL)`,
		`INSERT INTO migrations (migration, batch) VALUES ('2014_10_12_000000_create_users_table', 1)`,
		`CREATE TABLE users (
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
		)`,
		`INSERT INTO users (id, name, email, password, role, created_at, updated_at)
		 VALUES (1, 'Ada', 'ada@example.com', '$2y$10$legacyhash', 'admin', '2026-01-02 14:39:00', '2026-01-02 14:39:00')`,
		`CREATE TABLE currencies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			symbol TEXT NOT NULL,
			decimals INTEGER NOT NULL DEFAULT 2,
			is_base INTEGER NOT NULL DEFAULT 0,
			rate REAL NOT NULL DEFAULT 1,
			created_at TEXT,
			updated_at TEXT
		)`,
		`INSERT INTO currencies (id, code, name, symbol, decimals, is_base, rate) VALUES (1, 'USD', 'US Dollar', '$', 2, 1, 1)`,
		`CREATE TABLE accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			currency_id INTEGER NOT NULL,
			initial_balance REAL NOT NULL DEFAULT 0,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT,
			updated_at TEXT
		)`,
		`INSERT INTO accounts (id, name, type, currency_id, initial_balance, is_active) VALUES (1, 'Cash', 'cash', 1, 100, 1)`,
		`CREATE TABLE transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			account_id INTEGER NOT NULL,
			to_account_id INTEGER,
			category_id INTEGER,
			amount REAL NOT NULL,
			to_amount REAL,
			exchange_rate REAL,
			description TEXT,
			date TEXT,
			created_at TEXT,
			updated_at TEXT
		)`,
		`INSERT INTO transactions (id, type, account_id, amount, description, date)
		 VALUES (1, 'expense', 1, 12.5, 'Coffee', '2026-01-03')`,
		`INSERT INTO currencies (id, code, name, symbol, decimals, is_base, rate) VALUES (2, 'JPY', 'Yen', 'Y', 0, 0, 0.0067)`,
		`INSERT INTO currencies (id, code, name, symbol, decimals, is_base, rate) VALUES (3, 'BTC', 'Bitcoin', 'B', 8, 0, 50000)`,
		`INSERT INTO accounts (id, name, type, currency_id, initial_balance, is_active) VALUES (2, 'Yen', 'cash', 2, 3000, 1)`,
		`INSERT INTO accounts (id, name, type, currency_id, initial_balance, is_active) VALUES (3, 'Wallet', 'cash', 3, 0.5, 1)`,
		`INSERT INTO transactions (id, type, account_id, amount, description, date)
		 VALUES (2, 'expense', 2, 1500, 'Ramen', '2026-01-04')`,
		`INSERT INTO transactions (id, type, account_id, amount, description, date)
		 VALUES (3, 'income', 3, 0.00012345, 'Mining', '2026-01-05')`,
		`INSERT INTO transactions (id, type, account_id, to_account_id, amount, to_amount, exchange_rate, description, date)
		 VALUES (4, 'transfer', 1, 2, 10.5, 1567, 149.2, 'Exchange', '2026-01-06')`,
		`CREATE TABLE transaction_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			transaction_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			quantity REAL NOT NULL DEFAULT 1,
			price_per_unit REAL NOT NULL,
			total_price REAL NOT NULL
		)`,
		`INSERT INTO transaction_items (id, transaction_id, name, quantity, price_per_unit, total_price)
		 VALUES (1, 1, 'Beans', 5, 2.5, 12.5)`,
		`CREATE TABLE budgets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			amount REAL NOT NULL,
			period TEXT NOT NULL
		)`,
		`INSERT INTO budgets (id, name, amount, period) VALUES (1, 'Food', 750.5, 'monthly')`,
		`CREATE TABLE jobs (id INTEGER PRIMARY KEY, queue TEXT)`,
	}
	for _, s := range stmts {
		if _, err := sqlDB.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
}

func assertCopied(t *testing.T, dest *sql.DB) {
	t.Helper()
	assertCount(t, dest, "users", 1)
	assertCount(t, dest, "currencies", 3)
	assertCount(t, dest, "accounts", 3)
	assertCount(t, dest, "transactions", 4)
	assertMinorUnits(t, dest)
	var name, code, desc string
	if err := dest.QueryRow(`SELECT name FROM users`).Scan(&name); err != nil || name != "Ada" {
		t.Fatalf("user %q %v", name, err)
	}
	if err := dest.QueryRow(`SELECT code FROM currencies WHERE id = 1`).Scan(&code); err != nil || code != "USD" {
		t.Fatalf("currency %q %v", code, err)
	}
	if err := dest.QueryRow(`SELECT description FROM transactions`).Scan(&desc); err != nil || desc != "Coffee" {
		t.Fatalf("tx %q %v", desc, err)
	}
}

func assertCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != want {
		t.Fatalf("%s count %d want %d", table, n, want)
	}
}

// assertMinorUnits checks Laravel major-unit money became integer minor units
// scaled by each currency (USD 2, JPY 0, BTC 8) and that budgets got a currency.
func assertMinorUnits(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	checks := []struct {
		query string
		want  int64
	}{
		{`SELECT initial_balance FROM accounts WHERE id = 1`, 10000},
		{`SELECT initial_balance FROM accounts WHERE id = 2`, 3000},
		{`SELECT initial_balance FROM accounts WHERE id = 3`, 50000000},
		{`SELECT amount FROM transactions WHERE id = 1`, 1250},
		{`SELECT amount FROM transactions WHERE id = 2`, 1500},
		{`SELECT amount FROM transactions WHERE id = 3`, 12345},
		{`SELECT amount FROM transactions WHERE id = 4`, 1050},
		{`SELECT to_amount FROM transactions WHERE id = 4`, 1567},
		{`SELECT price_per_unit FROM transaction_items WHERE id = 1`, 250},
		{`SELECT total_price FROM transaction_items WHERE id = 1`, 1250},
		{`SELECT amount FROM budgets WHERE id = 1`, 75050},
		{`SELECT currency_id FROM budgets WHERE id = 1`, 1},
	}
	for _, c := range checks {
		var got int64
		if err := sqlDB.QueryRow(c.query).Scan(&got); err != nil {
			t.Fatalf("%s: %v", c.query, err)
		}
		if got != c.want {
			t.Errorf("%s = %d, want %d", c.query, got, c.want)
		}
	}
}

func TestUpgradeInPlaceLeavesGoSchemaUsable(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "database.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	createLaravelShape(t, sqlDB)
	if err := EnsureColumns(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	if err := migrate.Up(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	if err := UpgradeInPlace(ctx, sqlDB, ""); err != nil {
		t.Fatal(err)
	}

	var typ string
	if err := sqlDB.QueryRow(`SELECT type FROM pragma_table_info('transactions') WHERE name = 'amount'`).Scan(&typ); err != nil || typ != "INTEGER" {
		t.Fatalf("transactions.amount type = %q, %v; want INTEGER", typ, err)
	}
	var idx int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'transactions_account_type_date_amount_idx'`).Scan(&idx); err != nil || idx != 1 {
		t.Fatalf("amount index restored = %d, %v", idx, err)
	}

	btc, err := (domain.Transactions{DB: sqlDB}).ByID(ctx, 3)
	if err != nil || btc == nil || !btc.Amount.Equal(decimal.RequireFromString("0.00012345")) {
		t.Fatalf("btc tx via domain = %+v, %v", btc, err)
	}
	acct, err := (domain.Accounts{DB: sqlDB}).ByID(ctx, 3)
	if err != nil || acct == nil || !acct.Balance.Equal(decimal.RequireFromString("0.50012345")) {
		t.Fatalf("btc account via domain = %+v, %v", acct, err)
	}
}
