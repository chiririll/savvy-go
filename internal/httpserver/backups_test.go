package httpserver

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"savvy-go/internal/auth"
	"savvy-go/internal/db"
	"savvy-go/internal/legacy"
)

// TestRestoreLegacyLaravelBackup verifies that restoring a genuine PHP/Laravel
// -era backup through the HTTP restore endpoint runs the legacy column
// catch-up and upgrade-in-place steps, not just Go schema migrations, so the
// restored database ends up fully usable (e.g. categories.is_default present).
func TestRestoreLegacyLaravelBackup(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)
	ctx := context.Background()

	legacyPath := filepath.Join(t.TempDir(), "legacy.sqlite")
	legacyDB, err := db.Open(legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	loadLaravelFixture(t, legacyDB)
	if err := legacyDB.Close(); err != nil {
		t.Fatal(err)
	}

	backup, err := a.s.backups.Ingest(ctx, legacyPath, nil)
	if err != nil {
		t.Fatal(err)
	}

	res := a.do("POST", "/api/backups/"+backup.Filename+"/restore", nil, sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("restore %d %v", res.StatusCode, body)
	}
	// Restore swaps in a new *sql.DB (a.s.db); newTestApp's cleanup only
	// closes the original connection it captured, so close this one too
	// or Windows can't clean up the temp dir (file still in use).
	t.Cleanup(func() { _ = a.s.db.Close() })

	// The server's live connection now points at the restored (former
	// Laravel) file. Confirm the Go-only column made it in via legacy
	// column catch-up, not just Go's own CREATE TABLE IF NOT EXISTS.
	var isDefault int
	if err := a.s.db.QueryRowContext(ctx,
		`SELECT is_default FROM categories WHERE name = 'Groceries'`,
	).Scan(&isDefault); err != nil {
		t.Fatalf("is_default column missing after restore: %v", err)
	}

	if !legacy.AlreadyImported(ctx, a.s.db) {
		t.Fatal("expected legacy import stamp after restore")
	}

	// Regression: Laravel stored dates as "YYYY-MM-DD 00:00:00" in columns
	// declared date, which the API returned as a timestamp, so the edit form's
	// date input came up empty and saving wiped the date.
	tx, err := a.s.txs.ByID(ctx, 1)
	if err != nil || tx == nil || tx.Date == nil || *tx.Date != "2026-01-03" {
		t.Fatalf("restored transaction date = %+v, %v; want 2026-01-03", tx, err)
	}
}

// TestRestoreOldLaravelBackupKeepsLiveDB verifies that a Laravel backup older
// than legacy.LatestMigration is listed as unsupported and rejected
// by restore without touching the live database or its connection.
func TestRestoreOldLaravelBackupKeepsLiveDB(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)
	ctx := context.Background()

	oldPath := filepath.Join(t.TempDir(), "old.sqlite")
	oldDB, err := db.Open(oldPath)
	if err != nil {
		t.Fatal(err)
	}
	loadLaravelFixture(t, oldDB)
	if _, err := oldDB.Exec(`DELETE FROM migrations WHERE migration = ?`, legacy.LatestMigration); err != nil {
		t.Fatal(err)
	}
	if err := oldDB.Close(); err != nil {
		t.Fatal(err)
	}
	backup, err := a.s.backups.Ingest(ctx, oldPath, nil)
	if err != nil {
		t.Fatal(err)
	}

	if got := listedBackup(t, a, sess.Token, backup.Filename); got["status"] != "legacyUnsupported" || got["restorable"] != false {
		t.Fatalf("listed %v, want legacyUnsupported and not restorable", got)
	}

	res := a.do("POST", "/api/backups/"+backup.Filename+"/restore", nil, sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("restore %d %v", res.StatusCode, body)
	}

	// Live connection still works and still holds the original data.
	var email string
	if err := a.s.db.QueryRowContext(ctx, `SELECT email FROM users WHERE id = ?`, u.ID).Scan(&email); err != nil || email != "rw@test.com" {
		t.Fatalf("live db damaged: %q %v", email, err)
	}
}

// TestBackupStatus verifies the listed status follows the backup's schema
// migrations and that restore refuses a backup made by a newer app.
func TestBackupStatus(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("rw@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)
	ctx := context.Background()

	create := func(edit string) string {
		t.Helper()
		b, err := a.s.backups.Create(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if edit != "" {
			f, err := sql.Open("sqlite", a.s.backups.Path(*b))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			if _, err := f.Exec(edit); err != nil {
				t.Fatal(err)
			}
		}
		return b.Filename
	}
	current := create("")
	outdated := create(`DELETE FROM schema_migrations WHERE version = (SELECT MAX(version) FROM schema_migrations)`)
	newer := create(`INSERT INTO schema_migrations (version, applied_at) VALUES ('9999_from_the_future', '')`)

	for name, want := range map[string]string{current: "current", outdated: "outdated", newer: "newer"} {
		b := listedBackup(t, a, sess.Token, name)
		if b["status"] != want {
			t.Errorf("%s status %v, want %s", name, b["status"], want)
		}
		if b["restorable"] != (want != "newer") {
			t.Errorf("%s restorable %v", name, b["restorable"])
		}
		if wantPending := map[bool]float64{true: 1, false: 0}[want == "outdated"]; b["pendingCount"] != wantPending {
			t.Errorf("%s pendingCount %v, want %v", name, b["pendingCount"], wantPending)
		}
	}

	res := a.do("POST", "/api/backups/"+newer+"/restore", nil, sess.Token, sess.CSRF)
	if body := decodeJSON(t, res); res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("restore newer %d %v", res.StatusCode, body)
	}
}

// listedBackup returns the backup named name from GET /api/backups.
func listedBackup(t *testing.T, a *testApp, token, name string) map[string]any {
	t.Helper()
	res := a.do("GET", "/api/backups", nil, token, "")
	body := decodeJSON(t, res)
	list, _ := body["data"].([]any)
	for _, item := range list {
		if b, _ := item.(map[string]any); b["filename"] == name {
			return b
		}
	}
	t.Fatalf("backup %s not listed: %v", name, body)
	return nil
}

// loadLaravelFixture loads testdata/laravel.sql and records the latest
// supported Laravel migration so the database counts as up to date.
func loadLaravelFixture(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	script, err := os.ReadFile(filepath.Join("..", "..", "testdata", "laravel.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(string(script)); err != nil {
		t.Fatalf("load laravel.sql: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO migrations (migration, batch) VALUES (?, 2)`, legacy.LatestMigration); err != nil {
		t.Fatal(err)
	}
}
