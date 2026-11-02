package migrate

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"savvy-go/internal/db"
)

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

func TestUpAppliesEachMigrationOnce(t *testing.T) {
	for _, set := range []Set{Server, Space} {
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
			var mode string
			if err := sqlDB.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil || mode != "wal" {
				t.Fatalf("journal_mode %q, %v", mode, err)
			}
		})
	}
}

// UpTo stops after the named migration and a later Up finishes the rest.
func TestUpToStopsAtTheNamedMigration(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "space.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := Space.UpTo(ctx, sqlDB, BaseVersion); err != nil {
		t.Fatal(err)
	}
	applied, err := appliedVersions(ctx, sqlDB)
	if err != nil || !applied[BaseVersion] {
		t.Fatalf("the base is not applied: %v, %v", applied, err)
	}
	if err := Space.Up(ctx, sqlDB); err != nil {
		t.Fatalf("Up after UpTo: %v", err)
	}
	if n, err := Space.PendingCount(ctx, sqlDB); err != nil || n != 0 {
		t.Errorf("pending = %d, %v", n, err)
	}
}
