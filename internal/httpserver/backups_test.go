package httpserver

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
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
	createLaravelCategoriesFixture(t, legacyDB)
	if err := legacyDB.Close(); err != nil {
		t.Fatal(err)
	}

	backup, err := a.s.backups.Ingest(ctx, legacyPath, nil)
	if err != nil {
		t.Fatal(err)
	}

	res := a.do("POST", fmt.Sprintf("/api/backups/%d/restore", backup.ID), nil, sess.Token, sess.CSRF)
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
}

func createLaravelCategoriesFixture(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE migrations (id INTEGER PRIMARY KEY AUTOINCREMENT, migration TEXT NOT NULL, batch INTEGER NOT NULL)`,
		`INSERT INTO migrations (migration, batch) VALUES ('2014_10_12_000000_create_users_table', 1)`,
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			password TEXT,
			created_at TEXT,
			updated_at TEXT
		)`,
		`INSERT INTO users (id, name, email, password) VALUES (1, 'Ada', 'ada@example.com', '$2y$10$legacyhash')`,
		`CREATE TABLE categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			icon TEXT,
			color TEXT,
			created_at TEXT,
			updated_at TEXT
		)`,
		`INSERT INTO categories (id, name, type) VALUES (1, 'Groceries', 'expense')`,
		`CREATE TABLE jobs (id INTEGER PRIMARY KEY, queue TEXT)`,
	}
	for _, s := range stmts {
		if _, err := sqlDB.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
}
