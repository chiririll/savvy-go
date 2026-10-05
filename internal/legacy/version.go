package legacy

import (
	"context"
	"database/sql"
	"errors"
)

// LatestMigration is the last Laravel migration the importer was written
// against. Older Laravel databases may differ in ways the importer does not
// handle (column types, missing tables), so they are rejected up front rather
// than half-imported.
const LatestMigration = "2026_09_09_180000_restore_fractional_transaction_item_quantity"

var ErrUnsupportedVersion = errors.New("laravel database is older than the supported version; update the Laravel app to its latest version and create a new backup")

// ErrNotLaravel is returned for a file that is not a Laravel database.
var ErrNotLaravel = errors.New("not a laravel database")

// checkVersion accepts a Laravel database at LatestMigration.
func checkVersion(ctx context.Context, db *sql.DB) error {
	if !tableExists(ctx, db, "migrations") {
		return ErrNotLaravel
	}
	var found int
	if err := db.QueryRowContext(ctx, `SELECT 1 FROM migrations WHERE migration = ? LIMIT 1`, LatestMigration).Scan(&found); err != nil {
		return ErrUnsupportedVersion
	}
	return nil
}
