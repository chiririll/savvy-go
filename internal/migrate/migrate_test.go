package migrate

import (
	"context"
	"path/filepath"
	"testing"

	"savvy-go/internal/db"
)

func TestUpCreatesDomainTables(t *testing.T) {
	dir := t.TempDir()
	sqlDB, err := db.Open(filepath.Join(dir, "database.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	ctx := context.Background()
	pending, err := PendingCount(ctx, sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	if pending != -1 {
		t.Fatalf("pending before migrate: %d", pending)
	}

	if err := Up(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	if err := Up(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}

	pending, err = PendingCount(ctx, sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	if pending != 0 {
		t.Fatalf("pending after migrate: %d", pending)
	}

	required := []string{
		"users", "auth_sessions", "currencies", "accounts", "categories", "tags",
		"transactions", "transaction_items", "transaction_tag", "budgets",
		"recurring_transactions", "debts_placeholder_skip",
		"automation_rules", "settings", "uploads", "transaction_imports",
		"two_factor_challenges", "webauthn_credentials",
		"identity_providers", "password_tokens",
	}
	for _, table := range required {
		if table == "debts_placeholder_skip" {
			continue // debts are accounts with type=debt
		}
		var name string
		err := sqlDB.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Fatalf("missing table %s: %v", table, err)
		}
	}

	var mode string
	if err := sqlDB.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Fatalf("journal_mode %q", mode)
	}
}

// TestTablesAreStrict keeps the schema from drifting back to loosely typed
// tables: stored money must stay integer, never a float that slipped through.
func TestTablesAreStrict(t *testing.T) {
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "database.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	ctx := context.Background()
	if err := Up(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}

	rows, err := sqlDB.QueryContext(ctx, `SELECT name FROM pragma_table_list WHERE schema = 'main' AND type = 'table' AND strict = 0 AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		t.Errorf("table %s is not STRICT", name)
	}
}
