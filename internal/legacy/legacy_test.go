package legacy

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"slices"
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

func TestUnsupportedLaravelVersionRejected(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "database.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	createLaravelShape(t, sqlDB)
	if _, err := sqlDB.Exec(`DELETE FROM migrations WHERE migration = ?`, LatestMigration); err != nil {
		t.Fatal(err)
	}
	if info := Inspect(ctx, sqlDB); !info.Laravel || info.Supported {
		t.Fatalf("inspect %+v", info)
	}
	if err := Upgrade(ctx, sqlDB, ""); err != ErrUnsupportedVersion {
		t.Fatalf("Upgrade err %v", err)
	}
	if err := Copy(ctx, sqlDB, sqlDB); err != ErrUnsupportedVersion {
		t.Fatalf("Copy err %v", err)
	}
	if AlreadyImported(ctx, sqlDB) {
		t.Fatal("rejected database must not be stamped")
	}
}

func TestUpgrade(t *testing.T) {
	ctx := context.Background()
	sqlDB := upgradedFixture(t)

	// Second run only re-applies (already applied) migrations.
	if err := Upgrade(ctx, sqlDB, ""); err != nil {
		t.Fatal(err)
	}
	if !AlreadyImported(ctx, sqlDB) {
		t.Fatal("expected stamp")
	}
	if migrate.TableExists(ctx, sqlDB, "migrations") {
		t.Fatal("laravel migrations table should be dropped")
	}
	if cols, _ := migrate.Columns(ctx, sqlDB, "transactions"); slices.Contains(cols, "exchange_rate") {
		t.Fatal("transactions.exchange_rate should be dropped")
	}
	assertMinorUnits(t, sqlDB)
	assertDateOnly(t, sqlDB)
	var email string
	if err := sqlDB.QueryRow(`SELECT email FROM users WHERE id = 1`).Scan(&email); err != nil {
		t.Fatal(err)
	}
	if email != "ada@example.com" {
		t.Fatalf("email %q", email)
	}
}

func TestUpgradeLeavesGoSchemaUsable(t *testing.T) {
	ctx := context.Background()
	sqlDB := upgradedFixture(t)

	var typ string
	if err := sqlDB.QueryRow(`SELECT type FROM pragma_table_info('transactions') WHERE name = 'amount'`).Scan(&typ); err != nil || typ != "INTEGER" {
		t.Fatalf("transactions.amount type = %q, %v; want INTEGER", typ, err)
	}
	var idx int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'transactions_account_type_date_amount_idx'`).Scan(&idx); err != nil || idx != 1 {
		t.Fatalf("amount index restored = %d, %v", idx, err)
	}

	btc, err := (domain.Transactions{DB: sqlDB}).ByID(ctx, 3)
	if err != nil || btc == nil || !btc.Amount.Decimal().Equal(decimal.RequireFromString("0.00012345")) {
		t.Fatalf("btc tx via domain = %+v, %v", btc, err)
	}
	if btc.Date == nil || *btc.Date != "2026-01-05" {
		t.Fatalf("btc tx date = %v, want 2026-01-05", btc.Date)
	}
	acct, err := (domain.Accounts{DB: sqlDB}).ByID(ctx, 3)
	if err != nil || acct == nil || !acct.Balance.Decimal().Equal(decimal.RequireFromString("0.50012345")) {
		t.Fatalf("btc account via domain = %+v, %v", acct, err)
	}
}

// upgradedFixture loads the Laravel fixture and runs Upgrade on it, as the
// restore flow does.
func upgradedFixture(t *testing.T) *sql.DB {
	t.Helper()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "database.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	createLaravelShape(t, sqlDB)
	if err := Upgrade(context.Background(), sqlDB, ""); err != nil {
		t.Fatal(err)
	}
	return sqlDB
}

// baseCurrencyID is the base currency (EUR) in testdata/laravel.sql, which is
// deliberately not the lowest id.
const baseCurrencyID = 2

// createLaravelShape loads testdata/laravel.sql and records LatestMigration so
// the database counts as an up-to-date Laravel one.
func createLaravelShape(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	script, err := os.ReadFile(filepath.Join("..", "..", "testdata", "laravel.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(string(script)); err != nil {
		t.Fatalf("load laravel.sql: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO migrations (migration, batch) VALUES (?, 2)`, LatestMigration); err != nil {
		t.Fatal(err)
	}
}

func assertCopied(t *testing.T, dest *sql.DB) {
	t.Helper()
	assertCount(t, dest, "users", 1)
	assertCount(t, dest, "currencies", 4)
	assertCount(t, dest, "categories", 1)
	assertCount(t, dest, "accounts", 4)
	assertCount(t, dest, "recurring_transactions", 1)
	assertCount(t, dest, "transactions", 5)
	assertCount(t, dest, "transaction_items", 2)
	assertCount(t, dest, "budgets", 2)
	assertMinorUnits(t, dest)
	assertDateOnly(t, dest)
	var name, code, desc, status string
	if err := dest.QueryRow(`SELECT name FROM users`).Scan(&name); err != nil || name != "Ada" {
		t.Fatalf("user %q %v", name, err)
	}
	if err := dest.QueryRow(`SELECT code FROM currencies WHERE is_base = 1`).Scan(&code); err != nil || code != "EUR" {
		t.Fatalf("base currency %q %v", code, err)
	}
	if err := dest.QueryRow(`SELECT description FROM transactions WHERE id = 1`).Scan(&desc); err != nil || desc != "Coffee" {
		t.Fatalf("tx %q %v", desc, err)
	}
	if err := dest.QueryRow(`SELECT status FROM transactions WHERE id = 5`).Scan(&status); err != nil || status != "pending" {
		t.Fatalf("pending tx status %q %v", status, err)
	}
}

// assertDateOnly checks Laravel "YYYY-MM-DD 00:00:00" dates were cut to
// YYYY-MM-DD in every date column.
func assertDateOnly(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	var date string
	if err := sqlDB.QueryRow(`SELECT date FROM transactions WHERE id = 1`).Scan(&date); err != nil || date != "2026-01-03" {
		t.Fatalf("transactions.date %q %v", date, err)
	}
	for table, cols := range dateColumns {
		for _, c := range cols {
			var long int
			q := `SELECT COUNT(*) FROM ` + table + ` WHERE length(` + quoteIdent(c) + `) > 10`
			if err := sqlDB.QueryRow(q).Scan(&long); err != nil || long != 0 {
				t.Errorf("%s.%s has %d dates with a time part (%v)", table, c, long, err)
			}
		}
	}
	checks := []struct{ query, want string }{
		{`SELECT next_run_date FROM recurring_transactions WHERE id = 1`, "2026-10-15"},
		{`SELECT last_run_date FROM recurring_transactions WHERE id = 1`, "2026-09-15"},
		{`SELECT due_date FROM accounts WHERE id = 4`, "2027-08-15"},
		{`SELECT end_date FROM budgets WHERE id = 2`, "2026-08-31"},
	}
	for _, c := range checks {
		var got string
		if err := sqlDB.QueryRow(c.query).Scan(&got); err != nil || got != c.want {
			t.Errorf("%s = %q (%v), want %q", c.query, got, err, c.want)
		}
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
// scaled by each currency (USD/EUR 2, JPY 0, BTC 8), that a budget without a
// currency got the base currency, and that quantities stayed fractional.
func assertMinorUnits(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	checks := []struct {
		query string
		want  int64
	}{
		{`SELECT initial_balance FROM accounts WHERE id = 1`, 10000},
		{`SELECT initial_balance FROM accounts WHERE id = 2`, 3000},
		{`SELECT initial_balance FROM accounts WHERE id = 3`, 50000000},
		{`SELECT target_amount FROM accounts WHERE id = 4`, 120050},
		{`SELECT amount FROM transactions WHERE id = 1`, 1250},
		{`SELECT amount FROM transactions WHERE id = 2`, 1500},
		{`SELECT amount FROM transactions WHERE id = 3`, 12345},
		{`SELECT amount FROM transactions WHERE id = 4`, 1050},
		{`SELECT to_amount FROM transactions WHERE id = 4`, 1567},
		{`SELECT amount FROM transactions WHERE id = 5`, 1500},
		{`SELECT amount FROM recurring_transactions WHERE id = 1`, 1500},
		{`SELECT price_per_unit FROM transaction_items WHERE id = 1`, 250},
		{`SELECT total_price FROM transaction_items WHERE id = 1`, 1250},
		{`SELECT price_per_unit FROM transaction_items WHERE id = 2`, 3000},
		{`SELECT total_price FROM transaction_items WHERE id = 2`, 1500},
		{`SELECT amount FROM budgets WHERE id = 1`, 75050},
		{`SELECT currency_id FROM budgets WHERE id = 1`, baseCurrencyID},
		{`SELECT amount FROM budgets WHERE id = 2`, 30000},
		{`SELECT currency_id FROM budgets WHERE id = 2`, baseCurrencyID},
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
	var qty float64
	if err := sqlDB.QueryRow(`SELECT quantity FROM transaction_items WHERE id = 2`).Scan(&qty); err != nil || qty != 0.5 {
		t.Errorf("fractional quantity = %v (%v), want 0.5", qty, err)
	}
}
