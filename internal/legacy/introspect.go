package legacy

import (
	"context"
	"database/sql"

	"savvy-go/internal/migrate"
	"savvy-go/internal/migrate/upgrade"
)

// querier is the read side shared by *sql.DB, *sql.Tx and *sql.Conn.
type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// tableExists reports whether the main schema has a table called name.
func tableExists(ctx context.Context, q querier, name string) bool {
	var found string
	err := q.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&found)
	return err == nil
}

// columns lists the column names of table; empty when the table is missing.
func columns(ctx context.Context, q querier, table string) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT name FROM pragma_table_info(?)`, table)
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

// upBoth gives one file the server schema and the space schema of the oldest
// release: a Laravel database holds everything until it is split, and the
// conform code is written against that shape. Its space tables are converted
// to the current schema afterwards, like any old space
// (internal/migrate/upgrade).
func upBoth(ctx context.Context, db *sql.DB) error {
	if err := migrate.Server.Up(ctx, db); err != nil {
		return err
	}
	return upgrade.Entry().Apply(ctx, db)
}

// ensureIndexes recreates the indexes of both schemas, which schema surgery
// on a Laravel file drops together with the columns they covered.
func ensureIndexes(ctx context.Context, db *sql.DB) error {
	if err := migrate.Server.EnsureIndexes(ctx, db); err != nil {
		return err
	}
	return upgrade.Entry().EnsureIndexes(ctx, db)
}
