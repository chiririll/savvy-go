package upgrade_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"savvy-go/internal/db"
	"savvy-go/internal/migrate"
	"savvy-go/internal/migrate/upgrade"
	v1 "savvy-go/internal/migrate/upgrade/v1"
)

func space(t *testing.T, prepare func(*sql.DB)) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "space.sqlite")
	d, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	prepare(d)
	return path
}

func TestFindTellsAnOldFileFromACurrentOne(t *testing.T) {
	ctx := context.Background()
	old := space(t, func(d *sql.DB) {
		if err := v1.Apply(ctx, d); err != nil {
			t.Fatal(err)
		}
	})
	current := space(t, func(d *sql.DB) {
		if err := migrate.Space.Up(ctx, d); err != nil {
			t.Fatal(err)
		}
	})
	if hop, err := upgrade.Find(ctx, old); err != nil || hop == nil || hop.Version != v1.Version {
		t.Errorf("Find(old) = %v, %v", hop, err)
	}
	if hop, err := upgrade.Find(ctx, current); err != nil || hop != nil {
		t.Errorf("Find(current) = %v, %v; want nil", hop, err)
	}
}

func TestConvertTakesTheOldFileToTheCurrentSchema(t *testing.T) {
	ctx := context.Background()
	old := space(t, func(d *sql.DB) {
		if err := v1.Apply(ctx, d); err != nil {
			t.Fatal(err)
		}
	})
	dst := filepath.Join(t.TempDir(), "new.sqlite")
	if err := upgrade.Convert(ctx, old, dst); err != nil {
		t.Fatal(err)
	}
	if hop, err := upgrade.Find(ctx, dst); err != nil || hop != nil {
		t.Errorf("the result is still old: %v, %v", hop, err)
	}
	if err := upgrade.Convert(ctx, dst, filepath.Join(t.TempDir(), "again.sqlite")); err == nil {
		t.Error("a current file was converted")
	}
	if err := upgrade.Convert(ctx, old, dst); err == nil {
		t.Error("an existing target was overwritten")
	}
}

// The store asks only this: is the base recorded? A file that was never
// migrated is empty, not old; a file with space migrations but no base is.
func TestNeededLooksOnlyAtTheBase(t *testing.T) {
	ctx := context.Background()
	empty := space(t, func(*sql.DB) {})
	old := space(t, func(d *sql.DB) {
		if err := v1.Apply(ctx, d); err != nil {
			t.Fatal(err)
		}
	})
	current := space(t, func(d *sql.DB) {
		if err := migrate.Space.Up(ctx, d); err != nil {
			t.Fatal(err)
		}
	})
	server := space(t, func(d *sql.DB) {
		if err := migrate.Server.Up(ctx, d); err != nil {
			t.Fatal(err)
		}
	})
	for name, want := range map[string]struct {
		path string
		need bool
	}{"empty": {empty, false}, "old": {old, true}, "current": {current, false}, "server": {server, false}} {
		if got, err := upgrade.Needed(ctx, want.path); err != nil || got != want.need {
			t.Errorf("Needed(%s) = %v, %v; want %v", name, got, err, want.need)
		}
	}
}

// A file that is not at the base and that no hop knows is refused by name.
func TestConvertRefusesAnUnknownSchema(t *testing.T) {
	ctx := context.Background()
	odd := space(t, func(d *sql.DB) {
		for _, stmt := range []string{
			`CREATE TABLE schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL) STRICT`,
			`INSERT INTO schema_migrations VALUES ('space/0000_mystery', '2026-01-01T00:00:00Z')`,
		} {
			if _, err := d.Exec(stmt); err != nil {
				t.Fatal(err)
			}
		}
	})
	if need, err := upgrade.Needed(ctx, odd); err != nil || !need {
		t.Fatalf("Needed = %v, %v", need, err)
	}
	err := upgrade.Convert(ctx, odd, filepath.Join(t.TempDir(), "new.sqlite"))
	if !errors.Is(err, upgrade.ErrUnknownSchema) {
		t.Errorf("Convert = %v, want ErrUnknownSchema", err)
	}
}
