package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"regexp"
	"strings"

	_ "modernc.org/sqlite"
)

// Constraints (STRICT, CHECK, COLLATE, foreign key actions) cannot be added to
// an existing SQLite table, so a database created before they were declared
// keeps its old definitions. Conform rebuilds such tables to the current
// definitions. Each table is rebuilt by the procedure SQLite documents for
// schema changes: create the new table, copy the rows, drop the old one,
// rename.
//
// The target definitions come from running the embedded migrations on a
// scratch database, so they stay right when later migrations ALTER a table.
// A table needs a rebuild when it is not STRICT while the target is; a future
// constraint change that does not touch STRICT needs its own marker here.
//
// Columns the target table does not have (retired Go columns, Laravel-only
// columns) are dropped; rows that violate a constraint fail the whole run,
// which is rolled back as one transaction.

var createTableRE = regexp.MustCompile(`(?is)^\s*CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?("[^"]+"|\w+)\s*\(`)

type tableDef struct {
	name   string
	ddl    string
	strict bool
}

// Conform rebuilds every table that predates the current constraints and
// recreates the declared indexes. It does nothing for a Laravel database that
// is not converted yet (its money columns are still REAL), so run it after
// the legacy conversion. A conforming database costs one metadata query.
func Conform(ctx context.Context, db *sql.DB) error {
	if tableExists(ctx, db, "migrations") {
		return nil
	}
	target, err := targetTables(ctx)
	if err != nil {
		return err
	}
	var stale []tableDef
	for _, t := range target {
		if !t.strict {
			continue
		}
		strict, found, err := isStrict(ctx, db, t.name)
		if err != nil {
			return err
		}
		if found && !strict {
			stale = append(stale, t)
		}
	}
	if len(stale) > 0 {
		if err := rebuild(ctx, db, stale); err != nil {
			return err
		}
	}
	return EnsureIndexes(ctx, db)
}

// targetTables applies every embedded migration to an empty in-memory database
// and returns the tables it ends up with.
func targetTables(ctx context.Context) ([]tableDef, error) {
	scratch, err := sql.Open("sqlite", "file::memory:")
	if err != nil {
		return nil, err
	}
	defer scratch.Close()
	scratch.SetMaxOpenConns(1) // an in-memory database lives per connection

	names, err := migrationFiles()
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		body, err := fs.ReadFile(files, "sql/"+name)
		if err != nil {
			return nil, err
		}
		if err := execScript(ctx, scratch, string(body)); err != nil {
			return nil, fmt.Errorf("scratch %s: %w", name, err)
		}
	}
	rows, err := scratch.QueryContext(ctx, `
		SELECT m.name, m.sql, t.strict
		FROM sqlite_master m JOIN pragma_table_list t ON t.name = m.name AND t.schema = 'main'
		WHERE m.type = 'table' AND m.name NOT LIKE 'sqlite_%' ORDER BY m.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []tableDef
	for rows.Next() {
		var t tableDef
		var strict int
		if err := rows.Scan(&t.name, &t.ddl, &strict); err != nil {
			return nil, err
		}
		t.strict = strict == 1
		out = append(out, t)
	}
	return out, rows.Err()
}

func isStrict(ctx context.Context, db *sql.DB, table string) (strict, found bool, err error) {
	var v int
	err = db.QueryRowContext(ctx, `SELECT strict FROM pragma_table_list WHERE name = ? AND schema = 'main'`, table).Scan(&v)
	if err == sql.ErrNoRows {
		return false, false, nil
	}
	return v == 1, err == nil, err
}

func rebuild(ctx context.Context, db *sql.DB, tables []tableDef) error {
	// foreign_keys cannot change inside a transaction, and a pinned connection
	// keeps the pragma and the transaction on the same SQLite handle. With
	// enforcement off, dropping a parent table leaves its children untouched.
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys=OFF`); err != nil {
		return err
	}
	defer func() { _, _ = conn.ExecContext(context.Background(), `PRAGMA foreign_keys=ON`) }()

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, t := range tables {
		if err := rebuildTable(ctx, tx, t); err != nil {
			return fmt.Errorf("conform %s: %w", t.name, err)
		}
		slog.Info("schema conformed", "table", t.name)
	}
	rows, err := tx.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return err
	}
	violated := rows.Next()
	_ = rows.Close()
	if violated {
		return fmt.Errorf("conform: foreign key violations after rebuild")
	}
	return tx.Commit()
}

func rebuildTable(ctx context.Context, tx *sql.Tx, t tableDef) error {
	staging := t.name + "__conform"
	m := createTableRE.FindStringSubmatchIndex(t.ddl)
	if m == nil {
		return fmt.Errorf("cannot parse definition")
	}
	ddl := `CREATE TABLE "` + staging + `" (` + t.ddl[m[1]:]
	if _, err := tx.ExecContext(ctx, ddl); err != nil {
		return err
	}
	oldCols, err := tableColumns(ctx, tx, t.name)
	if err != nil {
		return err
	}
	newCols, err := tableColumns(ctx, tx, staging)
	if err != nil {
		return err
	}
	var common []string
	for _, c := range newCols {
		for _, o := range oldCols {
			if c == o {
				common = append(common, `"`+c+`"`)
				break
			}
		}
	}
	list := strings.Join(common, ", ")
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO "%s" (%s) SELECT %s FROM "%s"`, staging, list, list, t.name)); err != nil {
		return err
	}

	// AUTOINCREMENT remembers the highest id ever used, which can exceed the
	// highest remaining id; carry it over so ids are never reused.
	var seq sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT seq FROM sqlite_sequence WHERE name = ?`, t.name).Scan(&seq)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`DROP TABLE "%s"`, t.name)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE "%s" RENAME TO "%s"`, staging, t.name)); err != nil {
		return err
	}
	if seq.Valid {
		res, err := tx.ExecContext(ctx, `UPDATE sqlite_sequence SET seq = MAX(seq, ?) WHERE name = ?`, seq.Int64, t.name)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			if _, err := tx.ExecContext(ctx, `INSERT INTO sqlite_sequence (name, seq) VALUES (?, ?)`, t.name, seq.Int64); err != nil {
				return err
			}
		}
	}
	return nil
}

func tableColumns(ctx context.Context, tx *sql.Tx, table string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT name FROM pragma_table_info(?)`, table)
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
