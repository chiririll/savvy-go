package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCreatesTheDirectoryAndTheSafetyPragmas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "deeper", "x.sqlite")
	sqlDB, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the file is created: %v", err)
	}
	var mode string
	var fk int
	if err := sqlDB.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil || mode != "wal" {
		t.Errorf("journal mode %q, %v", mode, err)
	}
	if err := sqlDB.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil || fk != 1 {
		t.Errorf("foreign keys %d, %v", fk, err)
	}
}

func TestForeignKeysAreEnforced(t *testing.T) {
	sqlDB, err := Open(filepath.Join(t.TempDir(), "x.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	for _, q := range []string{
		`CREATE TABLE parent (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE child (id INTEGER PRIMARY KEY, parent_id INTEGER NOT NULL REFERENCES parent(id))`,
	} {
		if _, err := sqlDB.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := sqlDB.Exec(`INSERT INTO child (parent_id) VALUES (42)`); err == nil {
		t.Error("a row pointing at nothing was stored")
	}
}

// Extra pragmas must reach every connection, not only the first the pool opens.
func TestExtraPragmasApply(t *testing.T) {
	sqlDB, err := Open(filepath.Join(t.TempDir(), "x.sqlite"), "max_page_count(50)")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	var pages int
	if err := sqlDB.QueryRow(`PRAGMA max_page_count`).Scan(&pages); err != nil || pages != 50 {
		t.Errorf("max_page_count %d, %v", pages, err)
	}
}

func TestOpenAcceptsARelativePath(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	sqlDB, err := Open("rel.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if _, err := os.Stat(filepath.Join(dir, "rel.sqlite")); err != nil {
		t.Errorf("the file lands where the path says: %v", err)
	}
}

func TestReadyAndCheckpoint(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := Open(filepath.Join(t.TempDir(), "x.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if err := Ready(ctx, sqlDB); err != nil {
		t.Errorf("Ready: %v", err)
	}
	if _, err := sqlDB.Exec(`CREATE TABLE t (id INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if err := Checkpoint(ctx, sqlDB); err != nil {
		t.Errorf("Checkpoint: %v", err)
	}
	_ = sqlDB.Close()
	if err := Ready(ctx, sqlDB); err == nil {
		t.Error("a closed database is not ready")
	}
}
