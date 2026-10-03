package legacy

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"savvy-go/internal/auth"
)

const importStampKey = "legacy_import_completed_at"

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

// Copy order respects foreign keys (parents first).
var copyTables = []string{
	"users",
	"currencies",
	"accounts",
	"categories",
	"tags",
	"settings",
	"identity_providers",
	"recurring_transactions",
	"transactions",
	"transaction_items",
	"transaction_tag",
	"recurring_transaction_tag",
	"budgets",
	"budget_category",
	"budget_tag",
	"automation_rules",
	"automation_rule_logs",
	"uploads",
	"transaction_imports",
	"auth_sessions",
	"password_tokens",
	"two_factor_challenges",
	"two_factor_recovery_codes",
	"webauthn_credentials",
	"webauthn_challenges",
	"user_identities",
	"sso_login_states",
	"sso_login_tickets",
}

// IsLaravel reports a Laravel-era database (migration tracker present).
func IsLaravel(ctx context.Context, db *sql.DB) bool {
	return tableExists(ctx, db, "migrations")
}

// AlreadyImported is true when this file was already upgraded or never Laravel.
func AlreadyImported(ctx context.Context, db *sql.DB) bool {
	if !tableExists(ctx, db, "settings") {
		return false
	}
	var v string
	err := db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, importStampKey).Scan(&v)
	return err == nil && v != ""
}

// UpgradeInPlace stamps a Laravel-era database.sqlite that already holds
// domain tables (same names) so Go migrations and later starts are no-ops.
// appKey is the Laravel APP_KEY used to decrypt legacy encrypted TOTP secrets.
func UpgradeInPlace(ctx context.Context, db *sql.DB, appKey string) error {
	if !IsLaravel(ctx, db) {
		return nil
	}
	if AlreadyImported(ctx, db) {
		return nil
	}
	if err := CheckSupported(ctx, db); err != nil {
		return err
	}
	if err := ensureSettings(ctx, db); err != nil {
		return err
	}
	if err := auth.UnwrapLegacyTOTPSecrets(ctx, db, appKey); err != nil {
		slog.Warn("could not unwrap legacy totp secrets", "err", err)
	}
	// Before convertMoneyInPlace: a failure after it but before stamp would
	// rerun the money conversion on next start and scale amounts twice.
	if err := normalizeDates(ctx, db); err != nil {
		return err
	}
	if err := convertMoneyInPlace(ctx, db); err != nil {
		return fmt.Errorf("convert money to minor units: %w", err)
	}
	if err := stamp(ctx, db); err != nil {
		return err
	}
	dropLaravelOnly(ctx, db)
	slog.Info("legacy laravel database marked imported")
	return nil
}

// Copy transfers rows from a Laravel-shaped source into a Go-schema dest.
// Idempotent: INSERT OR IGNORE on primary keys; sqlite_sequence is raised
// so new IDs cannot collide.
func Copy(ctx context.Context, dest, src *sql.DB) error {
	if !IsLaravel(ctx, src) {
		return fmt.Errorf("source is not a laravel-era database")
	}
	if err := CheckSupported(ctx, src); err != nil {
		return err
	}

	sc, err := loadScales(ctx, src)
	if err != nil {
		return fmt.Errorf("load currency scales: %w", err)
	}
	for _, table := range copyTables {
		if !tableExists(ctx, src, table) || !tableExists(ctx, dest, table) {
			continue
		}
		n, err := copyTable(ctx, dest, src, table, sc)
		if err != nil {
			return fmt.Errorf("copy %s: %w", table, err)
		}
		if n > 0 {
			slog.Info("legacy import copied table", "table", table, "rows", n)
		}
	}
	if err := copySequences(ctx, dest, src); err != nil {
		return err
	}
	if err := ensureSettings(ctx, dest); err != nil {
		return err
	}
	if err := normalizeDates(ctx, dest); err != nil {
		return err
	}
	return stamp(ctx, dest)
}

// Laravel date casts stored "YYYY-MM-DD 00:00:00"; Go compares these columns
// as plain YYYY-MM-DD strings.
var dateColumns = map[string][]string{
	"transactions":           {"date"},
	"recurring_transactions": {"start_date", "end_date", "next_run_date", "last_run_date"},
	"budgets":                {"start_date", "end_date"},
	"accounts":               {"due_date"},
}

func normalizeDates(ctx context.Context, db *sql.DB) error {
	for table, cols := range dateColumns {
		if !tableExists(ctx, db, table) {
			continue
		}
		have, err := columns(ctx, db, table)
		if err != nil {
			return err
		}
		for _, c := range intersect(cols, have) {
			q := fmt.Sprintf(`UPDATE %s SET %[2]s = substr(%[2]s, 1, 10) WHERE length(%[2]s) > 10`, table, quote(c))
			if _, err := db.ExecContext(ctx, q); err != nil {
				return fmt.Errorf("normalize %s.%s: %w", table, c, err)
			}
		}
	}
	return nil
}

func copyTable(ctx context.Context, dest, src *sql.DB, table string, sc *scales) (int, error) {
	srcCols, err := columns(ctx, src, table)
	if err != nil {
		return 0, err
	}
	destCols, err := columns(ctx, dest, table)
	if err != nil {
		return 0, err
	}
	common := intersect(srcCols, destCols)
	if len(common) == 0 {
		return 0, nil
	}

	// Laravel budgets may predate currency_id, which the Go schema requires;
	// fixMoney fills it with the base currency.
	selectCols := quoteAll(common)
	if table == "budgets" && !slices.Contains(common, "currency_id") {
		common = append(common, "currency_id")
		selectCols = append(selectCols, "NULL")
	}
	quoted := quoteAll(common)
	q := fmt.Sprintf("SELECT %s FROM %s", strings.Join(selectCols, ", "), table)
	rows, err := src.QueryContext(ctx, q)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	placeholders := strings.Repeat("?,", len(common))
	placeholders = strings.TrimSuffix(placeholders, ",")
	insert := fmt.Sprintf(
		"INSERT OR IGNORE INTO %s (%s) VALUES (%s)",
		table, strings.Join(quoted, ", "), placeholders,
	)

	tx, err := dest.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, insert)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	n := 0
	for rows.Next() {
		raw := make([]any, len(common))
		ptrs := make([]any, len(common))
		for i := range raw {
			ptrs[i] = &raw[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return 0, err
		}
		if err := sc.fixMoney(table, common, raw); err != nil {
			return 0, err
		}
		if _, err := stmt.ExecContext(ctx, raw...); err != nil {
			return 0, err
		}
		n++
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return n, tx.Commit()
}

func copySequences(ctx context.Context, dest, src *sql.DB) error {
	if !tableExists(ctx, src, "sqlite_sequence") || !tableExists(ctx, dest, "sqlite_sequence") {
		return nil
	}
	rows, err := src.QueryContext(ctx, `SELECT name, seq FROM sqlite_sequence`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var seq int64
		if err := rows.Scan(&name, &seq); err != nil {
			return err
		}
		if !tableExists(ctx, dest, name) {
			continue
		}
		// sqlite_sequence has no UNIQUE(name), so ON CONFLICT upserts are rejected.
		res, err := dest.ExecContext(ctx, `UPDATE sqlite_sequence SET seq = MAX(seq, ?) WHERE name = ?`, seq, name)
		if err == nil {
			if n, _ := res.RowsAffected(); n == 0 {
				_, err = dest.ExecContext(ctx, `INSERT INTO sqlite_sequence(name, seq) VALUES(?, ?)`, name, seq)
			}
		}
		if err != nil {
			return fmt.Errorf("sqlite_sequence %s: %w", name, err)
		}
	}
	return rows.Err()
}

func stamp(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO settings(key, value) VALUES(?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		importStampKey, time.Now().UTC().Format(time.RFC3339),
	)
	return err
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

func tableExists(ctx context.Context, db *sql.DB, name string) bool {
	var found string
	err := db.QueryRowContext(ctx,
		`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name,
	).Scan(&found)
	return err == nil
}

func columns(ctx context.Context, db *sql.DB, table string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
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
		out[i] = quote(c)
	}
	return out
}

func quote(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}
