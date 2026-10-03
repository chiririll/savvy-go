package legacy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
)

// LatestMigration is the last Laravel migration the importer was written
// against. Older Laravel databases may differ in ways the importer does not
// handle (column types, missing tables), so they are rejected up front rather
// than half-imported.
const LatestMigration = "2026_09_09_180000_restore_fractional_transaction_item_quantity"

var ErrUnsupportedVersion = errors.New("laravel database is older than the supported version; update the Laravel app to its latest version and create a new backup")

// Info describes whether a database file is an importable Laravel-era one.
type Info struct {
	Laravel   bool
	Supported bool
}

// Inspect classifies db. A Go-schema or already imported database is not
// Laravel and needs no import, so it is always supported.
func Inspect(ctx context.Context, db *sql.DB) Info {
	if !IsLaravel(ctx, db) || AlreadyImported(ctx, db) {
		return Info{Supported: true}
	}
	var found int
	err := db.QueryRowContext(ctx, `SELECT 1 FROM migrations WHERE migration = ? LIMIT 1`, LatestMigration).Scan(&found)
	return Info{Laravel: true, Supported: err == nil}
}

// CheckSupported returns ErrUnsupportedVersion for a Laravel database that
// does not contain LatestMigration.
func CheckSupported(ctx context.Context, db *sql.DB) error {
	if !Inspect(ctx, db).Supported {
		return ErrUnsupportedVersion
	}
	return nil
}

// InspectFile is Inspect for a database file, opened read-only.
func InspectFile(ctx context.Context, path string) (Info, error) {
	src, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		return Info{}, fmt.Errorf("open: %w", err)
	}
	defer src.Close()
	return Inspect(ctx, src), nil
}
