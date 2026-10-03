package legacy

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"savvy-go/internal/migrate"
)

// Laravel declared these columns as date and stored "YYYY-MM-DD 00:00:00". Go
// treats them as plain YYYY-MM-DD text.
var dateColumns = map[string][]string{
	"transactions":           {"date"},
	"recurring_transactions": {"start_date", "end_date", "next_run_date", "last_run_date"},
	"budgets":                {"start_date", "end_date"},
	"accounts":               {"due_date"},
}

// normalizeDates cuts stored dates to YYYY-MM-DD. It is enough for a
// Go-schema database, whose date columns are already declared TEXT.
func normalizeDates(ctx context.Context, db *sql.DB) error {
	return eachDateColumn(ctx, db, func(table, col string) error {
		q := fmt.Sprintf(`UPDATE %s SET %[2]s = substr(%[2]s, 1, 10) WHERE length(%[2]s) > 10`, table, quoteIdent(col))
		if _, err := db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("normalize %s.%s: %w", table, col, err)
		}
		return nil
	})
}

// retypeDateColumns makes the date columns of a Laravel table declared TEXT
// holding YYYY-MM-DD. The declared type matters, not just the stored text: the
// sqlite driver turns values of DATE/DATETIME columns into time.Time, so a
// column still declared "date" reads back as "2026-01-03T00:00:00Z" no matter
// how clean the stored value is. Like convertMoneyInPlace it swaps each column
// through a new one instead of rebuilding the table, which would cascade
// through foreign keys. Columns already declared TEXT are only normalized.
func retypeDateColumns(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	err = eachDateColumn(ctx, tx, func(table, col string) error {
		decl, err := declaredType(ctx, tx, table, col)
		if err != nil {
			return err
		}
		if strings.EqualFold(decl, "TEXT") {
			return nil
		}
		notNull, err := isNotNull(ctx, tx, table, col)
		if err != nil {
			return err
		}
		tmp := col + "__text"
		add := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s TEXT`, table, quoteIdent(tmp))
		if notNull {
			add += " NOT NULL DEFAULT ''"
		}
		set := fmt.Sprintf(`UPDATE %s SET %s = substr(CAST(%s AS TEXT), 1, 10) WHERE %s IS NOT NULL`,
			table, quoteIdent(tmp), quoteIdent(col), quoteIdent(col))
		for _, ddl := range []string{add, set} {
			if _, err := tx.ExecContext(ctx, ddl); err != nil {
				return fmt.Errorf("%s: %w", ddl, err)
			}
		}
		if err := dropIndexesOn(ctx, tx, table, []string{col}); err != nil {
			return err
		}
		for _, ddl := range []string{
			fmt.Sprintf(`ALTER TABLE %s DROP COLUMN %s`, table, quoteIdent(col)),
			fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN %s TO %s`, table, quoteIdent(tmp), quoteIdent(col)),
		} {
			if _, err := tx.ExecContext(ctx, ddl); err != nil {
				return fmt.Errorf("%s: %w", ddl, err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	// Indexes on the swapped columns were dropped; recreate every declared index.
	return migrate.EnsureIndexes(ctx, db)
}

// eachDateColumn calls fn for every date column that exists in q's database.
func eachDateColumn(ctx context.Context, q querier, fn func(table, col string) error) error {
	for table, cols := range dateColumns {
		have, err := columns(ctx, q, table)
		if err != nil {
			return err
		}
		for _, c := range intersect(cols, have) {
			if err := fn(table, c); err != nil {
				return err
			}
		}
	}
	return nil
}

type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func declaredType(ctx context.Context, q rowQuerier, table, col string) (string, error) {
	var typ string
	err := q.QueryRowContext(ctx, `SELECT type FROM pragma_table_info(?) WHERE name = ?`, table, col).Scan(&typ)
	return typ, err
}

func isNotNull(ctx context.Context, q rowQuerier, table, col string) (bool, error) {
	var notNull int
	err := q.QueryRowContext(ctx, `SELECT "notnull" FROM pragma_table_info(?) WHERE name = ?`, table, col).Scan(&notNull)
	return notNull == 1, err
}
