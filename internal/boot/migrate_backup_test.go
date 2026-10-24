package boot

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/backup"
	"savvy-go/internal/signing"
	"savvy-go/internal/store/sqlite"
)

// dropAll stands in for a destructive migration (think 9999_drop): it removes
// every table and records its own version.
func dropAll(ctx context.Context, d *sql.DB) error {
	conn, err := d.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys=OFF`); err != nil {
		return err
	}
	rows, err := conn.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return err
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		tables = append(tables, name)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, name := range tables {
		if _, err := conn.ExecContext(ctx, `DROP TABLE "`+name+`"`); err != nil {
			return err
		}
	}
	_, err = conn.ExecContext(ctx, `CREATE TABLE schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL) STRICT;
		INSERT INTO schema_migrations VALUES ('server/9999_drop', 'now')`)
	return err
}

func dropOpts(ctx context.Context, cfg appconfig.Config) sqlite.Options {
	opts := sqlite.AppOptions(ctx, cfg.DataDir)
	opts.MigrateServer, opts.MigrateSpace = dropAll, dropAll
	return opts
}

func hasSpaces(ctx context.Context, st *sqlite.Store) bool {
	var n int
	_ = st.Server().QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE name='spaces'`).Scan(&n)
	return n > 0
}

func backupCount(cfg appconfig.Config) int {
	entries, _ := os.ReadDir(filepath.Join(cfg.BackupsDir, "server"))
	return len(entries)
}

// oldData is a data dir at the current schema, with a space row, that looks
// like it was written by an older release: its migration record is gone, so
// the current migrations count as pending.
func oldData(t *testing.T) appconfig.Config {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	cfg := appconfig.Config{DataDir: filepath.Join(dir, "data"), BackupsDir: filepath.Join(dir, "backups"), UploadsDir: filepath.Join(dir, "up")}
	if err := makeDirs(cfg); err != nil {
		t.Fatal(err)
	}
	st, err := sqlite.OpenApp(ctx, cfg.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSpace(ctx, 1); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO spaces (id, uuid, name, created_at, updated_at) VALUES (1, 'u-1', 'Home', 'now', 'now')`,
		`DELETE FROM schema_migrations`,
	} {
		if _, err := st.Server().ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := signing.Load(cfg.DataDir, false); err != nil { // the key exists once the server has run
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestPendingMigrationIsBackedUpFirstAndRestorable(t *testing.T) {
	ctx := context.Background()
	cfg := oldData(t)

	st, err := openMigrated(ctx, cfg, appconfig.Build{Version: "9.9"}, dropOpts(ctx, cfg))
	if err != nil {
		t.Fatal(err)
	}
	migrated := !hasSpaces(ctx, st)
	_ = st.Close()
	if !migrated {
		t.Fatal("the live database was not migrated")
	}
	if n := backupCount(cfg); n != 1 {
		t.Fatalf("want one backup, got %d", n)
	}

	// The backup holds the data from before and restores the normal way (on a
	// store without migrators, so the restored schema stays as it was).
	plain, err := sqlite.Open(ctx, sqlite.Options{Dir: cfg.DataDir})
	if err != nil {
		t.Fatal(err)
	}
	defer plain.Close()
	key, err := signing.Load(cfg.DataDir, true)
	if err != nil {
		t.Fatal(err)
	}
	svc := backup.Service{Store: plain, Keys: signing.KeyRing{Holder: signing.NewHolder(key)}, Dir: cfg.BackupsDir, DataDir: cfg.DataDir}
	list, err := svc.ServerBackups()
	if err != nil || len(list) != 1 || list[0].Note == nil || !strings.HasPrefix(*list[0].Note, "Before migration") {
		t.Fatalf("list %+v, err %v", list, err)
	}
	if err := svc.RestoreServer(ctx, list[0].Filename); err != nil {
		t.Fatalf("restore: %v", err)
	}
	var name string
	if err := plain.Server().QueryRowContext(ctx, `SELECT name FROM spaces WHERE id = 1`).Scan(&name); err != nil || name != "Home" {
		t.Fatalf("restored space %q, err %v", name, err)
	}
}

func TestFailedBackupStopsStartAndKeepsData(t *testing.T) {
	ctx := context.Background()
	cfg := oldData(t)
	// A file where the backup directory should be makes the backup fail.
	if err := os.WriteFile(filepath.Join(cfg.BackupsDir, "server"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if st, err := openMigrated(ctx, cfg, appconfig.Build{}, dropOpts(ctx, cfg)); err == nil {
		_ = st.Close()
		t.Fatal("the start went on without its backup")
	}
	st, err := sqlite.Open(ctx, sqlite.Options{Dir: cfg.DataDir}) // as on disk, no migrators
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if !hasSpaces(ctx, st) {
		t.Fatal("the data changed although the backup failed")
	}
}

func failOpts(ctx context.Context, cfg appconfig.Config) sqlite.Options {
	opts := sqlite.AppOptions(ctx, cfg.DataDir)
	fail := func(context.Context, *sql.DB) error { return errors.New("boom") }
	opts.MigrateServer, opts.MigrateSpace = fail, fail
	return opts
}

func TestRestartLoopReusesTheBackupUntilDataChanges(t *testing.T) {
	ctx := context.Background()
	cfg := oldData(t)

	for i := range 3 { // a migration that keeps failing under a restart loop
		if st, err := openMigrated(ctx, cfg, appconfig.Build{}, failOpts(ctx, cfg)); err == nil {
			_ = st.Close()
			t.Fatalf("start %d: the failing migration went through", i)
		}
	}
	if n := backupCount(cfg); n != 1 {
		t.Fatalf("restart loop made %d backups, want 1", n)
	}

	// Data written in between (an older release ran meanwhile) is not in the
	// reused backup, so the next attempt takes a new one.
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(cfg.DataDir, "server.sqlite"), later, later); err != nil {
		t.Fatal(err)
	}
	if st, err := openMigrated(ctx, cfg, appconfig.Build{}, failOpts(ctx, cfg)); err == nil {
		_ = st.Close()
		t.Fatal("the failing migration went through")
	}
	if n := backupCount(cfg); n != 2 {
		t.Fatalf("want a new backup after the data changed, got %d", n)
	}
}

func TestFreshOrCurrentDataMakesNoBackup(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	cfg := appconfig.Config{DataDir: filepath.Join(dir, "data"), BackupsDir: filepath.Join(dir, "backups"), UploadsDir: filepath.Join(dir, "up")}
	if err := makeDirs(cfg); err != nil {
		t.Fatal(err)
	}
	for range 2 { // first run, then a start with everything current
		st, err := openMigrated(ctx, cfg, appconfig.Build{}, sqlite.AppOptions(ctx, cfg.DataDir))
		if err != nil {
			t.Fatal(err)
		}
		_ = st.Close()
	}
	if n := backupCount(cfg); n != 0 {
		t.Fatalf("%d backups", n)
	}
}
