package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Open creates the parent directory, opens SQLite with WAL and a single
// connection (one writer), and applies the usual pragmas. Extra pragmas
// (e.g. "max_page_count(2560)") go into the DSN, so every connection the pool
// opens gets them, not only the first.
func Open(path string, pragmas ...string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o775); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dsn := sqliteDSN(path)
	for _, p := range pragmas {
		dsn += "&_pragma=" + p
	}
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(0)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if err := applyPragmas(ctx, sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return sqlDB, nil
}

// ReadOnlyURI is the read-only URI of a database file, for ATTACH or for
// looking into a file that must not change.
func ReadOnlyURI(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	abs = filepath.ToSlash(abs)
	if !strings.HasPrefix(abs, "/") {
		abs = "/" + abs
	}
	return "file:" + abs + "?mode=ro"
}

// OpenReadOnly opens an existing database file without changing it: no
// directory is made, no pragma is written, and the file is not created.
func OpenReadOnly(path string) (*sql.DB, error) {
	return sql.Open("sqlite", ReadOnlyURI(path))
}

func sqliteDSN(path string) string {
	abs := path
	if !filepath.IsAbs(path) {
		if resolved, err := filepath.Abs(path); err == nil {
			abs = resolved
		}
	}
	// modernc.org/sqlite uses a URI; forward slashes are required on Windows.
	abs = filepath.ToSlash(abs)
	if !strings.HasPrefix(abs, "/") && len(abs) > 1 && abs[1] == ':' {
		abs = "/" + abs
	}
	return "file:" + abs + "?_pragma=busy_timeout(5000)&_time_format=sqlite"
}

func applyPragmas(ctx context.Context, sqlDB *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
		"PRAGMA temp_store=MEMORY",
		"PRAGMA cache_size=-20000",
	}
	for _, p := range pragmas {
		if _, err := sqlDB.ExecContext(ctx, p); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
	}
	return nil
}

// Ready reports whether the database accepts a simple query.
func Ready(ctx context.Context, sqlDB *sql.DB) error {
	var n int
	return sqlDB.QueryRowContext(ctx, "SELECT 1").Scan(&n)
}

// Checkpoint folds the WAL into the main database file (backup-safe copy).
func Checkpoint(ctx context.Context, sqlDB *sql.DB) error {
	_, err := sqlDB.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)")
	return err
}

// RemoveFiles deletes a database file with its write-ahead log and journal.
func RemoveFiles(path string) {
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		_ = os.Remove(path + suffix)
	}
}
