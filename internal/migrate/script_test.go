package migrate

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"savvy-go/internal/db"
)

func TestSplitSQL(t *testing.T) {
	tests := []struct {
		name   string
		script string
		want   []string
	}{
		{"empty", "", nil},
		{"only comments and blanks", "-- hello\n\n  -- again\n", nil},
		{"one statement", "CREATE TABLE a (id INTEGER);\n", []string{"CREATE TABLE a (id INTEGER)"}},
		{"a statement over several lines", "CREATE TABLE a (\n  id INTEGER,\n  n TEXT\n);\n", []string{"CREATE TABLE a (\n  id INTEGER,\n  n TEXT\n)"}},
		{"two statements with a comment between", "SELECT 1;\n-- next\nSELECT 2;\n", []string{"SELECT 1", "SELECT 2"}},
		{"a last statement without a semicolon", "SELECT 1;\nSELECT 2", []string{"SELECT 1", "SELECT 2"}},
		{"a lone semicolon is nothing", ";\nSELECT 1;\n", []string{"SELECT 1"}},
	}
	for _, tc := range tests {
		got := splitSQL(tc.script)
		if strings.Join(got, "|") != strings.Join(tc.want, "|") {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestPreviewShortensLongStatements(t *testing.T) {
	if got := preview("SELECT   1\n  FROM   t"); got != "SELECT 1 FROM t" {
		t.Errorf("whitespace is collapsed: %q", got)
	}
	long := preview("SELECT " + strings.Repeat("x", 200))
	if !strings.HasSuffix(long, "…") || len([]rune(long)) != 81 {
		t.Errorf("long statement: %q", long)
	}
}

func TestBothSetsShareOneFileWithoutClashing(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "both.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := Server.Up(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	if pending, _ := Space.PendingCount(ctx, sqlDB); pending == 0 {
		t.Error("applying the server set must leave the space set pending")
	}
	if err := Space.Up(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version LIKE 'server/%'`).Scan(&n); err != nil || n == 0 {
		t.Errorf("server versions: %d, %v", n, err)
	}
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version LIKE 'space/%'`).Scan(&n); err != nil || n == 0 {
		t.Errorf("space versions: %d, %v", n, err)
	}
}

func TestVersionsListEachMigrationOnce(t *testing.T) {
	for _, set := range []Set{Server, Space} {
		versions, err := set.Versions()
		if err != nil || len(versions) == 0 {
			t.Fatalf("%s: %v, %v", set, versions, err)
		}
		for _, v := range versions {
			if !strings.HasPrefix(v, string(set)+"/") || strings.HasSuffix(v, ".sql") {
				t.Errorf("version %q", v)
			}
		}
	}
}

func TestEnsureIndexesRestoresADroppedIndex(t *testing.T) {
	ctx := context.Background()
	sqlDB := migrated(t, Space)
	var name string
	err := sqlDB.QueryRow(`SELECT name FROM sqlite_master WHERE type='index' AND name='transactions_account_dedup_unique'`).Scan(&name)
	if err != nil {
		t.Fatalf("the dedup index must exist after migrating: %v", err)
	}
	if _, err := sqlDB.Exec(`DROP INDEX transactions_account_dedup_unique`); err != nil {
		t.Fatal(err)
	}
	for range 2 { // idempotent
		if err := Space.EnsureIndexes(ctx, sqlDB); err != nil {
			t.Fatal(err)
		}
	}
	if err := sqlDB.QueryRow(`SELECT name FROM sqlite_master WHERE type='index' AND name='transactions_account_dedup_unique'`).Scan(&name); err != nil {
		t.Errorf("the index was not restored: %v", err)
	}
}

func TestAFailingScriptChangesNothing(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "x.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	err = execScript(ctx, sqlDB, "CREATE TABLE ok (id INTEGER);\nCREATE TABLE ok (id INTEGER);\n")
	if err == nil || !strings.Contains(err.Error(), "CREATE TABLE ok") {
		t.Fatalf("err = %v", err)
	}
	var n int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name='ok'`).Scan(&n); err != nil || n != 0 {
		t.Errorf("the first statement survived the failure of the second: %d, %v", n, err)
	}
}
