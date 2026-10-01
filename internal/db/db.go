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
// connection (one writer), and applies the usual pragmas.
func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o775); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dsn := sqliteDSN(path)
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
