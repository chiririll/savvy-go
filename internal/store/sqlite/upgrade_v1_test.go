package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"savvy-go/internal/db"
	v1 "savvy-go/internal/migrate/upgrade/v1"
	"savvy-go/internal/store"
)

// writeV1Space makes the file of a v1.0 space with one account and an expense.
func writeV1Space(t *testing.T, path string) {
	t.Helper()
	d, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	script := `CREATE TABLE schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL) STRICT;
INSERT INTO schema_migrations VALUES ('` + v1.Version + `', '2026-01-01T00:00:00Z');
` + v1.Schema + `
INSERT INTO currencies (id, code, name, symbol, is_base) VALUES (1, 'EUR', 'Euro', '€', 1);
INSERT INTO accounts (id, name, type, currency_id, initial_balance) VALUES (1, 'Card', 'bank', 1, 500);
INSERT INTO transactions (id, type, account_id, amount, date) VALUES (1, 'expense', 1, 120, '2026-02-01');
`
	for _, stmt := range v1.Statements(script) {
		if _, err := d.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
}

func TestOpenConvertsAV1Space(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, spacesDir, "3.sqlite")
	writeV1Space(t, path)

	s, err := OpenApp(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	space, err := s.Space(ctx, 3)
	if err != nil {
		t.Fatalf("the v1.0 space did not open: %v (%v)", err, s.Status())
	}
	if n := count(t, space, `SELECT COUNT(*) FROM payments WHERE account_id = 1 AND amount = -120`); n != 1 {
		t.Errorf("payments of the expense = %d, want 1", n)
	}
	if n := count(t, space, `SELECT COUNT(*) FROM schema_migrations WHERE version = 'space/0001_base'`); n != 1 {
		t.Errorf("the base is not recorded")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	// The v1.0 file stays as a backup, and a second start converts nothing.
	backup := path + oldBackup
	if is, err := v1.IsV1(ctx, backup); err != nil || !is {
		t.Errorf("the backup is not the v1.0 file: %v, %v", is, err)
	}
	s, err = OpenApp(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if ids, _ := s.Spaces(ctx); len(ids) != 1 || ids[0] != 3 {
		t.Errorf("the backup must not look like a space: %v", ids)
	}
}

// An uploaded v1.0 backup is converted on the way in: copying its columns into
// the current tables would leave the payments and lines out without an error.
func TestPrepareSpaceConvertsAV1Backup(t *testing.T) {
	ctx := context.Background()
	src := filepath.Join(t.TempDir(), "backup.sqlite")
	writeV1Space(t, src)

	s := openApp(t)
	prepared, err := s.PrepareSpace(ctx, src, store.WithSpaceUUID("sp-1"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ImportSpace(ctx, 4, prepared); err != nil {
		t.Fatal(err)
	}
	space, err := s.Space(ctx, 4)
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, space, `SELECT COUNT(*) FROM payments WHERE account_id = 1 AND amount = -120`); n != 1 {
		t.Errorf("payments of the expense = %d, want 1", n)
	}
	if n := count(t, space, `SELECT COUNT(*) FROM transaction_lines WHERE transaction_id = 1 AND amount = 120`); n != 1 {
		t.Errorf("lines of the expense = %d, want 1", n)
	}
	if n := count(t, space, `SELECT COUNT(*) FROM space_settings WHERE key = 'space_uuid' AND value = 'sp-1'`); n != 1 {
		t.Errorf("the space uuid was not written")
	}
}
