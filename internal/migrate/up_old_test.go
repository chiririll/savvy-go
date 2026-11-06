package migrate

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"savvy-go/internal/db"
)

// recording opens a file that records the given space migrations and nothing
// else, as a file of an older release does.
func recording(t *testing.T, versions ...string) (*sql.DB, func()) {
	t.Helper()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "old.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(tableSQL); err != nil {
		t.Fatal(err)
	}
	for _, v := range versions {
		if _, err := sqlDB.Exec(`INSERT INTO schema_migrations (version, applied_at) VALUES (?, '2026-01-01T00:00:00Z')`, v); err != nil {
			t.Fatal(err)
		}
	}
	return sqlDB, func() { _ = sqlDB.Close() }
}

// A file of an older release records space migrations that are not the base.
// The base uses CREATE TABLE IF NOT EXISTS, which would skip the old tables, so
// Up must refuse whichever release it is, instead of leaving half a schema.
func TestUpRefusesAnyOlderSpaceFile(t *testing.T) {
	ctx := context.Background()
	for _, versions := range [][]string{
		{"space/0001_initial"},
		{"space/0001_initial", "space/0002_something"},
		{"space/0000_unheard_of"},
	} {
		sqlDB, closeDB := recording(t, versions...)

		err := Space.Up(ctx, sqlDB)
		if !errors.Is(err, ErrOldSchema) {
			t.Errorf("Up on %v = %v, want ErrOldSchema", versions, err)
		} else if !strings.Contains(err.Error(), "converted") {
			t.Errorf("the message must say what to do: %v", err)
		}
		var n int
		if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name = 'payments'`).Scan(&n); err != nil || n != 0 {
			t.Errorf("a refused Up must change nothing: %d, %v", n, err)
		}
		if old, err := IsOldSpace(ctx, sqlDB); err != nil || !old {
			t.Errorf("IsOldSpace(%v) = %v, %v", versions, old, err)
		}
		closeDB()
	}
}

// A file at the base, one never migrated and one holding only server
// migrations are not old, and Server.Up must not trip over old space versions
// in a shared file.
func TestIsOldSpaceOnlyLooksAtSpaceMigrations(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		versions []string
		old      bool
	}{
		"base":       {[]string{BaseVersion}, false},
		"base+later": {[]string{BaseVersion, "space/0002_later"}, false},
		"server":     {[]string{"server/0001_initial"}, false},
		"none":       {nil, false},
		"old":        {[]string{"space/0001_initial"}, true},
	} {
		sqlDB, closeDB := recording(t, tc.versions...)
		if old, err := IsOldSpace(ctx, sqlDB); err != nil || old != tc.old {
			t.Errorf("%s: IsOldSpace = %v, %v; want %v", name, old, err, tc.old)
		}
		closeDB()
	}

	sqlDB, closeDB := recording(t, "space/0001_initial")
	defer closeDB()
	if err := Server.Up(ctx, sqlDB); err != nil {
		t.Fatalf("Server.Up = %v", err)
	}
}
