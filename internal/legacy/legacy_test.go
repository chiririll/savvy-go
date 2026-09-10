package legacy

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"savvy-go/internal/db"
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
	if err := UpgradeInPlace(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	if err := UpgradeInPlace(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	if !AlreadyImported(ctx, sqlDB) {
		t.Fatal("expected stamp")
	}
	if tableExists(ctx, sqlDB, "migrations") {
		t.Fatal("laravel migrations table should be dropped")
	}
	var email string
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
	assertCount(t, dest, "currencies", 1)
	assertCount(t, dest, "accounts", 1)
	assertCount(t, dest, "transactions", 1)
	var name, code, desc string
	if err := dest.QueryRow(`SELECT name FROM users`).Scan(&name); err != nil || name != "Ada" {
		t.Fatalf("user %q %v", name, err)
	}
	if err := dest.QueryRow(`SELECT code FROM currencies`).Scan(&code); err != nil || code != "USD" {
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
