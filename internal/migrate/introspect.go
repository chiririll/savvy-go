package migrate

import (
	"context"
	"database/sql"
)

// Querier is the read side shared by *sql.DB, *sql.Tx and *sql.Conn.
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// TableExists reports whether the main schema has a table called name.
func TableExists(ctx context.Context, q Querier, name string) bool {
	var found string
	err := q.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&found)
	return err == nil
}

// Columns lists the column names of table; empty when the table is missing.
func Columns(ctx context.Context, q Querier, table string) ([]string, error) {
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
