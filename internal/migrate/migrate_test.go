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

// TestConformRebuildsOldTables degrades a table to the pre-STRICT shape, with
// an extra retired column, and expects Conform to restore the current
// definition without losing rows or the id sequence.
func TestConformRebuildsOldTables(t *testing.T) {
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "database.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	ctx := context.Background()
	if err := Up(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	exec := func(q string) {
		t.Helper()
		if _, err := sqlDB.ExecContext(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	exec(`PRAGMA foreign_keys=OFF`)
	exec(`CREATE TABLE tags_old (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE, created_at TEXT, updated_at TEXT, retired TEXT)`)
	exec(`DROP TABLE tags`)
	exec(`ALTER TABLE tags_old RENAME TO tags`)
	exec(`PRAGMA foreign_keys=ON`)
	exec(`INSERT INTO tags (id, name, retired) VALUES (1, 'a', 'x'), (2, 'b', 'y'), (3, 'c', 'z')`)
	exec(`DELETE FROM tags WHERE id = 3`) // the sequence stays at 3

	if err := Conform(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	if err := Conform(ctx, sqlDB); err != nil { // idempotent
		t.Fatal(err)
	}

	var strict, retired int
	if err := sqlDB.QueryRowContext(ctx, `SELECT strict FROM pragma_table_list WHERE name = 'tags' AND schema = 'main'`).Scan(&strict); err != nil || strict != 1 {
		t.Fatalf("tags strict = %d, %v", strict, err)
	}
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('tags') WHERE name = 'retired'`).Scan(&retired); err != nil || retired != 0 {
		t.Fatalf("retired column kept (%d, %v)", retired, err)
	}
	var n int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM tags WHERE name IN ('a', 'b')`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("rows kept = %d, %v", n, err)
	}
	res, err := sqlDB.ExecContext(ctx, `INSERT INTO tags (name) VALUES ('d')`)
	if err != nil {
		t.Fatal(err)
	}
	if id, _ := res.LastInsertId(); id != 4 {
		t.Errorf("next id = %d, want 4 (sequence carried over)", id)
	}
	var enforced int
	if err := sqlDB.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&enforced); err != nil || enforced != 1 {
		t.Errorf("foreign_keys left %d, %v", enforced, err)
	}
}
