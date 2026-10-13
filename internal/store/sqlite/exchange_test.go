package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"savvy-go/internal/db"
	"savvy-go/internal/legacy"
	"savvy-go/internal/migrate"
	"savvy-go/internal/store"
)

func openApp(t *testing.T) *Store {
	t.Helper()
	s, err := OpenApp(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func exec(t *testing.T, d store.DB, q string, args ...any) {
	t.Helper()
	if _, err := d.ExecContext(context.Background(), q, args...); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
}

func count(t *testing.T, d store.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := d.QueryRowContext(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return n
}

// seedSpace gives a space a currency, an account and two tags, and deletes
// the second tag so the autoincrement counter is ahead of the rows.
func seedSpace(t *testing.T, d store.DB) {
	t.Helper()
	exec(t, d, `INSERT INTO currencies (code, name, symbol, decimals, is_base, rate) VALUES ('EUR', 'Euro', '€', 2, 1, '1')`)
	exec(t, d, `INSERT INTO accounts (name, type, currency_id) VALUES ('Cash', 'cash', 1)`)
	exec(t, d, `INSERT INTO tags (name) VALUES ('keep'), ('gone')`)
	exec(t, d, `DELETE FROM tags WHERE name = 'gone'`)
	exec(t, d, `INSERT INTO space_settings (key, value) VALUES ('space_uuid', 'uuid-1')`)
}

// P19, P26: a space backup comes back with the same ids, and ids freed before
// the backup are not handed out again.
func TestP19SpaceRoundTripKeepsIDs(t *testing.T) {
	ctx := context.Background()
	s := openApp(t)
	_ = s.CreateSpace(ctx, 1)
	d, _ := s.Space(ctx, 1)
	seedSpace(t, d)
	backup := filepath.Join(t.TempDir(), "space.sqlite")
	if err := s.ExportSpace(ctx, 1, backup); err != nil {
		t.Fatal(err)
	}
	exec(t, d, `DELETE FROM accounts`)

	p, err := s.PrepareSpace(ctx, backup)
	if err != nil {
		t.Fatal(err)
	}
	if p.UUID != "uuid-1" || p.Size <= 0 {
		t.Fatalf("prepared %+v", p)
	}
	if err := s.ReplaceSpace(ctx, 1, p); err != nil {
		t.Fatal(err)
	}
	if n := count(t, d, `SELECT COUNT(*) FROM accounts WHERE id = 1 AND name = 'Cash'`); n != 1 {
		t.Fatal("the account did not come back with its id")
	}
	exec(t, d, `INSERT INTO tags (name) VALUES ('new')`)
	if n := count(t, d, `SELECT id FROM tags WHERE name = 'new'`); n != 3 {
		t.Fatalf("new tag got id %d, want 3 (2 was used before the backup)", n)
	}
	if _, err := os.Stat(p.Artifact); !os.IsNotExist(err) {
		t.Fatal("the prepared artifact must be removed once used")
	}
}

// P18: nothing of a backup's schema runs, and only known tables and columns
// are read.
func TestP18UntrustedSpaceFileIsSanitized(t *testing.T) {
	ctx := context.Background()
	s := openApp(t)
	_ = s.CreateSpace(ctx, 1)
	d, _ := s.Space(ctx, 1)
	seedSpace(t, d)
	backup := filepath.Join(t.TempDir(), "space.sqlite")
	if err := s.ExportSpace(ctx, 1, backup); err != nil {
		t.Fatal(err)
	}

	// Doctor the file: a trigger that would fire on our inserts, a view, an
	// extra table and an extra column.
	evil, err := db.Open(backup)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`CREATE TABLE stolen (x TEXT)`,
		`CREATE TRIGGER spy AFTER INSERT ON tags BEGIN INSERT INTO stolen VALUES (NEW.name); END`,
		`CREATE VIEW leak AS SELECT * FROM accounts`,
		`ALTER TABLE accounts ADD COLUMN evil TEXT`,
		`UPDATE accounts SET evil = 'x'`,
	} {
		if _, err := evil.Exec(q); err != nil {
			t.Fatal(q, err)
		}
	}
	_ = evil.Close()

	p, err := s.PrepareSpace(ctx, backup)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceSpace(ctx, 1, p); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"stolen", "spy", "leak"} {
		if n := count(t, d, `SELECT COUNT(*) FROM sqlite_master WHERE name = ?`, name); n != 0 {
			t.Fatalf("%s came along from the backup", name)
		}
	}
	if n := count(t, d, `SELECT COUNT(*) FROM pragma_table_info('accounts') WHERE name = 'evil'`); n != 0 {
		t.Fatal("an unknown column came along")
	}
	exec(t, d, `INSERT INTO tags (name) VALUES ('after')`)
	if n := count(t, d, `SELECT COUNT(*) FROM accounts`); n != 1 {
		t.Fatalf("accounts %d", n)
	}
}

// P18: files that are not usable space databases are refused.
func TestP18RejectsBadFiles(t *testing.T) {
	ctx := context.Background()
	s := openApp(t)
	dir := t.TempDir()

	garbage := filepath.Join(dir, "garbage.sqlite")
	_ = os.WriteFile(garbage, []byte(strings.Repeat("not a database ", 100)), 0o644)
	if _, err := s.PrepareSpace(ctx, garbage); err == nil {
		t.Fatal("garbage accepted")
	}

	// A server database is not a space backup.
	if _, err := s.PrepareSpace(ctx, filepath.Join(s.opts.Dir, serverFile)); !errors.Is(err, ErrWrongKind) {
		t.Fatalf("server file as space: %v", err)
	}

	// A file from a newer app is refused.
	_ = s.CreateSpace(ctx, 1)
	newer := filepath.Join(dir, "newer.sqlite")
	_ = s.ExportSpace(ctx, 1, newer)
	nd, _ := db.Open(newer)
	_, _ = nd.Exec(`INSERT INTO schema_migrations (version, applied_at) VALUES ('space/9999_future', '')`)
	_ = nd.Close()
	if _, err := s.PrepareSpace(ctx, newer); !errors.Is(err, ErrNewerSchema) {
		t.Fatalf("newer file: %v", err)
	}

	// Rows pointing at missing rows are refused.
	broken := filepath.Join(dir, "broken.sqlite")
	_ = s.ExportSpace(ctx, 1, broken)
	bd, _ := db.Open(broken)
	_, _ = bd.Exec(`PRAGMA foreign_keys = OFF`)
	if _, err := bd.Exec(`INSERT INTO accounts (name, type, currency_id) VALUES ('Orphan', 'cash', 99)`); err != nil {
		t.Fatal(err)
	}
	_ = bd.Close()
	if _, err := s.PrepareSpace(ctx, broken); err == nil || !strings.Contains(err.Error(), "pointing at rows") {
		t.Fatalf("dangling reference: %v", err)
	}
	if entries, _ := os.ReadDir(filepath.Join(s.opts.Dir, stagingDir)); len(entries) != 0 {
		t.Fatalf("failed prepares left %d staging dirs", len(entries))
	}
}

func TestImportSpaceCreatesSpace(t *testing.T) {
	ctx := context.Background()
	s := openApp(t)
	_ = s.CreateSpace(ctx, 1)
	d, _ := s.Space(ctx, 1)
	seedSpace(t, d)
	backup := filepath.Join(t.TempDir(), "space.sqlite")
	_ = s.ExportSpace(ctx, 1, backup)
	p, err := s.PrepareSpace(ctx, backup)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ImportSpace(ctx, 2, p); err != nil {
		t.Fatal(err)
	}
	d2, err := s.Space(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, d2, `SELECT COUNT(*) FROM accounts`); n != 1 {
		t.Fatalf("imported accounts %d", n)
	}
}

// P20: a server backup holds the server and every space and comes back whole.
func TestP20ServerRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := openApp(t)
	srv := s.Server()
	exec(t, srv, `INSERT INTO users (id, name, email, role) VALUES (1, 'Ada', 'ada@x', 'admin')`)
	for _, id := range []int64{1, 2} {
		exec(t, srv, `INSERT INTO spaces (id, uuid, name) VALUES (?, ?, ?)`, id, "u"+string(rune('0'+id)), "S")
		_ = s.CreateSpace(ctx, id)
	}
	d1, _ := s.Space(ctx, 1)
	seedSpace(t, d1)
	dir := t.TempDir()
	if err := s.ExportServer(ctx, dir); err != nil {
		t.Fatal(err)
	}

	exec(t, srv, `DELETE FROM users`)
	exec(t, d1, `DELETE FROM accounts`)
	_ = s.DeleteSpace(ctx, 2)

	p, err := s.PrepareServer(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceServer(ctx, p); err != nil {
		t.Fatal(err)
	}
	if n := count(t, s.Server(), `SELECT COUNT(*) FROM users`); n != 1 {
		t.Fatalf("users %d", n)
	}
	ids, _ := s.Spaces(ctx)
	if len(ids) != 2 {
		t.Fatalf("spaces %v", ids)
	}
	d1, _ = s.Space(ctx, 1)
	if n := count(t, d1, `SELECT COUNT(*) FROM accounts`); n != 1 {
		t.Fatalf("space 1 accounts %d", n)
	}
}

func TestServerBackupMustMatchItsSpaceFiles(t *testing.T) {
	ctx := context.Background()
	s := openApp(t)
	exec(t, s.Server(), `INSERT INTO spaces (id, uuid, name) VALUES (1, 'u', 'S'), (2, 'v', 'T')`)
	_ = s.CreateSpace(ctx, 1)
	_ = s.CreateSpace(ctx, 2)
	dir := t.TempDir()
	_ = s.ExportServer(ctx, dir)
	_ = os.Remove(filepath.Join(dir, spacesDir, "2.sqlite"))
	if _, err := s.PrepareServer(ctx, dir); err == nil {
		t.Fatal("a server backup missing a space file was accepted")
	}
}

// A Go database never holds the server and the finances in one file, so such
// a file is refused rather than split.
func TestGoFileWithEverythingIsRefused(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "database.sqlite")
	d, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, set := range []migrate.Set{migrate.Server, migrate.Space} {
		if err := set.Up(ctx, d); err != nil {
			t.Fatal(err)
		}
	}
	_ = d.Close()

	s := openApp(t)
	if _, err := s.PrepareServer(ctx, path); !errors.Is(err, ErrWrongKind) {
		t.Fatalf("as a server: %v", err)
	}
	if _, err := s.PrepareSpace(ctx, path); !errors.Is(err, ErrWrongKind) {
		t.Fatalf("as a space: %v", err)
	}
}

// P21: a Laravel backup is converted, then split into the server and space 1
// with the Laravel roles mapped; as a space backup only its finances are taken.
func TestP21LaravelBackup(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "laravel.sqlite")
	d, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "laravel.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(string(script)); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO migrations (migration, batch) VALUES ('` + legacy.LatestMigration + `', 2)`,
		`INSERT INTO users (id, name, email, role) VALUES (2, 'Writer', 'w@x', 'read-write'), (3, 'Reader', 'r@x', 'read-only')`,
		`CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT)`,
		`INSERT INTO settings (key, value) VALUES ('auto_update_currencies', 'false')`,
	} {
		if _, err := d.Exec(q); err != nil {
			t.Fatal(q, err)
		}
	}
	_ = d.Close()

	s := openApp(t)
	sp, err := s.PrepareSpace(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ImportSpace(ctx, 5, sp); err != nil {
		t.Fatal(err)
	}
	space, _ := s.Space(ctx, 5)
	if n := count(t, space, `SELECT COUNT(*) FROM transactions`); n != 5 {
		t.Fatalf("transactions %d", n)
	}
	if n := count(t, s.Server(), `SELECT COUNT(*) FROM users`); n != 0 {
		t.Fatal("a space import must not bring users")
	}

	p, err := s.PrepareServer(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceServer(ctx, p); err != nil {
		t.Fatal(err)
	}
	srv := s.Server()
	if n := count(t, srv, `SELECT COUNT(*) FROM users WHERE role IN ('admin', 'user')`); n != 3 {
		t.Fatalf("users with mapped roles %d", n)
	}
	for user, role := range map[int]string{1: "admin", 2: "editor", 3: "viewer"} {
		if n := count(t, srv, `SELECT COUNT(*) FROM space_members WHERE space_id = 1 AND user_id = ? AND role = ?`, user, role); n != 1 {
			t.Fatalf("user %d is not %s of space 1", user, role)
		}
	}
	space, _ = s.Space(ctx, 1)
	if n := count(t, space, `SELECT COUNT(*) FROM accounts`); n != 4 {
		t.Fatalf("accounts %d", n)
	}
	if n := count(t, space, `SELECT COUNT(*) FROM space_settings WHERE key = 'auto_update_currencies' AND value = 'false'`); n != 1 {
		t.Fatal("the space setting did not move into the space")
	}
	var uuid string
	_ = srv.QueryRowContext(ctx, `SELECT uuid FROM spaces WHERE id = 1`).Scan(&uuid)
	if n := count(t, space, `SELECT COUNT(*) FROM space_settings WHERE key = 'space_uuid' AND value = ?`, uuid); n != 1 {
		t.Fatal("the space file does not carry its registry uuid")
	}
}
