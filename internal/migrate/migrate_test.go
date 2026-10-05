package migrate

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"savvy-go/internal/db"
)

// setTables are tables each set must create.
var setTables = map[Set][]string{
	Server: {
		"users", "auth_sessions", "password_tokens", "two_factor_challenges", "webauthn_credentials",
		"api_tokens", "settings", "uploads", "identity_providers", "spaces", "space_members",
		"space_invitations", "space_links", "trusted_keys", "admin_audit",
	},
	Space: {
		"currencies", "accounts", "categories", "tags", "transactions", "transaction_items",
		"transaction_tag", "budgets", "recurring_transactions", "automation_rules",
		"transaction_imports", "space_transfers", "space_settings",
	},
}

// migrated opens a fresh file with set applied.
func migrated(t *testing.T, set Set) *sql.DB {
	t.Helper()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), string(set)+".sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := set.Up(context.Background(), sqlDB); err != nil {
		t.Fatal(err)
	}
	return sqlDB
}

func TestUpCreatesTables(t *testing.T) {
	for set, tables := range setTables {
		t.Run(string(set), func(t *testing.T) {
			ctx := context.Background()
			sqlDB, err := db.Open(filepath.Join(t.TempDir(), "database.sqlite"))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = sqlDB.Close() })

			if pending, err := set.PendingCount(ctx, sqlDB); err != nil || pending != -1 {
				t.Fatalf("pending before migrate: %d, %v", pending, err)
			}
			for range 2 { // a second run applies nothing
				if err := set.Up(ctx, sqlDB); err != nil {
					t.Fatal(err)
				}
			}
			if pending, err := set.PendingCount(ctx, sqlDB); err != nil || pending != 0 {
				t.Fatalf("pending after migrate: %d, %v", pending, err)
			}

			for _, table := range tables {
				var name string
				if err := sqlDB.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name); err != nil {
					t.Errorf("missing table %s: %v", table, err)
				}
			}
			var mode string
			if err := sqlDB.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil || mode != "wal" {
				t.Fatalf("journal_mode %q, %v", mode, err)
			}
		})
	}
}

// TestTablesAreStrict keeps the schema from drifting back to loosely typed
// tables: stored money must stay integer, never a float that slipped through.
func TestTablesAreStrict(t *testing.T) {
	for _, set := range []Set{Server, Space} {
		t.Run(string(set), func(t *testing.T) {
			sqlDB := migrated(t, set)
			rows, err := sqlDB.Query(`SELECT name FROM pragma_table_list WHERE schema = 'main' AND type = 'table' AND strict = 0 AND name NOT LIKE 'sqlite_%'`)
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
		})
	}
}

// TestForeignKeysAreIndexed requires an index led by every foreign key column.
// Without one, every lookup by that column and every delete of a parent row
// (the ON DELETE check or action) scans the whole child table.
func TestForeignKeysAreIndexed(t *testing.T) {
	for _, set := range []Set{Server, Space} {
		t.Run(string(set), func(t *testing.T) {
			sqlDB := migrated(t, set)
			query := func(q string, args ...any) []string {
				t.Helper()
				rows, err := sqlDB.Query(q, args...)
				if err != nil {
					t.Fatal(err)
				}
				defer rows.Close()
				var out []string
				for rows.Next() {
					var s string
					if err := rows.Scan(&s); err != nil {
						t.Fatal(err)
					}
					out = append(out, s)
				}
				return out
			}
			for _, table := range query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`) {
				leading := map[string]bool{}
				// A partial index only covers some rows, so it does not count.
				for _, idx := range query(`SELECT name FROM pragma_index_list(?) WHERE partial = 0`, table) {
					if first := query(`SELECT name FROM pragma_index_info(?) WHERE seqno = 0`, idx); len(first) == 1 {
						leading[first[0]] = true
					}
				}
				for _, pk := range query(`SELECT name FROM pragma_table_info(?) WHERE pk = 1`, table) {
					leading[pk] = true // INTEGER PRIMARY KEY or the first column of a composite key
				}
				for _, col := range query(`SELECT "from" FROM pragma_foreign_key_list(?)`, table) {
					if !leading[col] {
						t.Errorf("%s.%s is a foreign key without an index", table, col)
					}
				}
			}
		})
	}
}
