package migrate

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"
)

//go:embed sql/server/*.sql sql/space/*.sql
var files embed.FS

// Set is one family of migrations: the server database or a space database.
// Versions are recorded as "<set>/<file>", so both sets can be applied to one
// file without clashing.
type Set string

const (
	Server Set = "server"
	Space  Set = "space"
)

// Up applies the pending migrations of this set.
func (s Set) Up(ctx context.Context, db *sql.DB) error { return up(ctx, db, s) }

// EnsureIndexes creates the indexes this set declares.
func (s Set) EnsureIndexes(ctx context.Context, db *sql.DB) error { return ensureIndexes(ctx, db, s) }

// PendingCount is the number of this set's migrations not yet applied.
func (s Set) PendingCount(ctx context.Context, db *sql.DB) (int, error) {
	return pendingCount(ctx, db, s)
}

const tableSQL = `CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TEXT NOT NULL
) STRICT`

func up(ctx context.Context, db *sql.DB, sets ...Set) error {
	if _, err := db.ExecContext(ctx, tableSQL); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return err
	}

	names, err := migrationFiles(sets...)
	if err != nil {
		return err
	}

	for _, name := range names {
		version := strings.TrimSuffix(name, ".sql")
		if applied[version] {
			continue
		}
		body, err := fs.ReadFile(files, "sql/"+name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if err := execScript(ctx, db, string(body)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err := db.ExecContext(ctx,
			`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
			version, time.Now().UTC().Format(time.RFC3339),
		); err != nil {
			return fmt.Errorf("record %s: %w", name, err)
		}
	}
	return nil
}

func ensureIndexes(ctx context.Context, db *sql.DB, sets ...Set) error {
	names, err := migrationFiles(sets...)
	if err != nil {
		return err
	}
	for _, name := range names {
		body, err := fs.ReadFile(files, "sql/"+name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		for _, stmt := range splitSQL(string(body)) {
			up := strings.ToUpper(strings.TrimSpace(stmt))
			if !strings.HasPrefix(up, "CREATE INDEX") && !strings.HasPrefix(up, "CREATE UNIQUE INDEX") {
				continue
			}
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("%s: %w", preview(stmt), err)
			}
		}
	}
	return nil
}

// pendingCount is the number of the sets' migrations not yet recorded, or -1
// when schema_migrations is missing (never migrated).
func pendingCount(ctx context.Context, db *sql.DB, sets ...Set) (int, error) {
	var name string
	err := db.QueryRowContext(ctx,
		`SELECT name FROM sqlite_master WHERE type='table' AND name='schema_migrations'`,
	).Scan(&name)
	if err == sql.ErrNoRows {
		names, listErr := migrationFiles(sets...)
		if listErr != nil {
			return 0, listErr
		}
		if len(names) == 0 {
			return 0, nil
		}
		return -1, nil
	}
	if err != nil {
		return 0, err
	}

	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return 0, err
	}
	names, err := migrationFiles(sets...)
	if err != nil {
		return 0, err
	}
	pending := 0
	for _, name := range names {
		if !applied[strings.TrimSuffix(name, ".sql")] {
			pending++
		}
	}
	return pending, nil
}

func appliedVersions(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("list applied migrations: %w", err)
	}
	defer rows.Close()

	out := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

// migrationFiles lists the files of the sets as "<set>/<file>", each set in
// file order and the sets in the order given.
func migrationFiles(sets ...Set) ([]string, error) {
	var names []string
	for _, set := range sets {
		entries, err := fs.ReadDir(files, "sql/"+string(set))
		if err != nil {
			return nil, err
		}
		var own []string
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
				continue
			}
			own = append(own, string(set)+"/"+e.Name())
		}
		sort.Strings(own)
		names = append(names, own...)
	}
	return names, nil
}

func execScript(ctx context.Context, db *sql.DB, script string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, stmt := range splitSQL(script) {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("%s: %w", preview(stmt), err)
		}
	}
	return tx.Commit()
}

func splitSQL(script string) []string {
	var (
		out     []string
		current strings.Builder
	)
	for _, line := range strings.Split(script, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "--") {
			continue
		}
		current.WriteString(line)
		current.WriteByte('\n')
		if strings.HasSuffix(trim, ";") {
			stmt := strings.TrimSpace(current.String())
			stmt = strings.TrimSuffix(stmt, ";")
			if stmt != "" {
				out = append(out, stmt)
			}
			current.Reset()
		}
	}
	if leftover := strings.TrimSpace(current.String()); leftover != "" {
		out = append(out, leftover)
	}
	return out
}

func preview(stmt string) string {
	stmt = strings.Join(strings.Fields(stmt), " ")
	if len(stmt) > 80 {
		return stmt[:80] + "…"
	}
	return stmt
}

// Versions lists the versions of this set's migrations, as recorded.
func (s Set) Versions() ([]string, error) {
	names, err := migrationFiles(s)
	if err != nil {
		return nil, err
	}
	for i, n := range names {
		names[i] = strings.TrimSuffix(n, ".sql")
	}
	return names, nil
}
