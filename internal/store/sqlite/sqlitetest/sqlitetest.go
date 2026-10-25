// Package sqlitetest opens migrated application stores for tests. The server
// migrations run once per test binary; each test gets a copy of the migrated
// file instead of migrating again, which costs about a tenth of the time.
package sqlitetest

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"savvy-go/internal/store/sqlite"
)

const serverFile = "server.sqlite"

var (
	once     sync.Once
	template []byte
	initErr  error
)

// migratedServer is the server database after the embedded migrations.
func migratedServer() ([]byte, error) {
	once.Do(func() {
		dir, err := os.MkdirTemp("", "sqlitetest-")
		if err != nil {
			initErr = err
			return
		}
		defer func() { _ = os.RemoveAll(dir) }()
		st, err := sqlite.OpenApp(context.Background(), dir)
		if err != nil {
			initErr = err
			return
		}
		// Closing folds the write-ahead log into the file.
		if initErr = st.Close(); initErr != nil {
			return
		}
		template, initErr = os.ReadFile(filepath.Join(dir, serverFile))
	})
	return template, initErr
}

// Prepare puts a migrated server database into dir, which must not hold a
// store yet. OpenApp on dir then finds the migrations applied.
func Prepare(t testing.TB, dir string) {
	t.Helper()
	data, err := migratedServer()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, serverFile), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

// Open is a store in a fresh temporary directory, closed when the test ends.
func Open(t testing.TB) *sqlite.Store { return OpenIn(t, t.TempDir()) }

// OpenIn is Open in dir, which must not hold a store yet.
func OpenIn(t testing.TB, dir string) *sqlite.Store {
	t.Helper()
	Prepare(t, dir)
	st, err := sqlite.OpenApp(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}
