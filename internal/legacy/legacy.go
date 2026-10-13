package legacy

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

)

// Laravel-only tables that the Go runtime does not use.
var laravelOnlyTables = []string{
	"migrations",
	"jobs",
	"failed_jobs",
	"job_batches",
	"cache",
	"cache_locks",
	"sessions",
}

// Upgrade converts a Laravel database file to the Go schema in place, with
// the server and the space tables still in one file for the caller to split.
// It runs once, on a staging copy: a failure leaves the copy to be thrown
// away. The file must be at LatestMigration. appKey is the Laravel APP_KEY,
// used to decrypt legacy encrypted TOTP secrets.
func Upgrade(ctx context.Context, db *sql.DB, appKey string) error {
	if err := checkVersion(ctx, db); err != nil {
		return err
	}
	if err := ensureColumns(ctx, db); err != nil {
		return fmt.Errorf("legacy columns: %w", err)
	}
	// Before the unique indexes and NOCASE columns see the data.
	if err := dedupeNames(ctx, db); err != nil {
		return err
	}
	if err := upBoth(ctx, db); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if err := convert(ctx, db, appKey); err != nil {
		return fmt.Errorf("legacy import: %w", err)
	}
	// The conversion keeps the Laravel table definitions; rebuild them to the
	// Go ones (STRICT, CHECK, NOCASE, no Laravel-only columns).
	if err := conform(ctx, db); err != nil {
		return fmt.Errorf("conform schema: %w", err)
	}
	return nil
}

// convert turns the Laravel data, which already sits in tables of the Go
// names, into the Go representation.
func convert(ctx context.Context, db *sql.DB, appKey string) error {
	if err := ensureSettings(ctx, db); err != nil {
		return err
	}
	unwrapped, reset, err := UpgradeLegacyTOTPSecrets(ctx, db, appKey)
	if err != nil {
		return fmt.Errorf("convert two-factor secrets: %w", err)
	}
	if unwrapped+reset > 0 {
		slog.Info("laravel two-factor secrets converted", "decrypted", unwrapped, "reset", reset)
	}
	if err := retypeDateColumns(ctx, db); err != nil {
		return fmt.Errorf("convert dates to text: %w", err)
	}
	if err := convertMoneyInPlace(ctx, db); err != nil {
		return fmt.Errorf("convert money to minor units: %w", err)
	}
	if err := mapRoles(ctx, db); err != nil {
		return fmt.Errorf("map roles: %w", err)
	}
	dropLaravelOnly(ctx, db)
	slog.Info("laravel database converted")
	return nil
}

func ensureSettings(ctx context.Context, db *sql.DB) error {
	if tableExists(ctx, db, "settings") {
		return nil
	}
	_, err := db.ExecContext(ctx, `CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT)`)
	return err
}

func dropLaravelOnly(ctx context.Context, db *sql.DB) {
	for _, table := range laravelOnlyTables {
		if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS `+table); err != nil {
			slog.Warn("could not drop laravel table", "table", table, "err", err)
		}
	}
}

func intersect(a, b []string) []string {
	have := make(map[string]bool, len(b))
	for _, c := range b {
		have[c] = true
	}
	var out []string
	for _, c := range a {
		if have[c] {
			out = append(out, c)
		}
	}
	return out
}

func quoteAll(cols []string) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = quoteIdent(c)
	}
	return out
}
