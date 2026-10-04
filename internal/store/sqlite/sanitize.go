package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"savvy-go/internal/db"
	"savvy-go/internal/migrate"
	"savvy-go/internal/store"
)

// A database file from outside (an uploaded or downloaded backup) is never
// opened as a live database: its schema could hold triggers or views that
// run our statements' side effects, and its tables could be shaped to break
// our constraints. Instead a fresh file is created from our migrations and
// only the rows of known tables and columns are copied into it, read through
// a read-only ATTACH with trusted_schema off. The fresh file's constraints
// and a foreign key check then validate the data.

// copyFile is the file a sanitize writes to; the caller removes it on error.
type copyPlan struct {
	set migrate.Set
	// transforms replaces a column's value by an SQL expression over the
	// source row, e.g. to map retired roles.
	transforms map[string]map[string]string
}

var errNotSQLite = errors.New("not a readable SQLite database")

// srcURI is the read-only URI of a source file for ATTACH.
func srcURI(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	abs = filepath.ToSlash(abs)
	if !strings.HasPrefix(abs, "/") {
		abs = "/" + abs
	}
	return "file:" + abs + "?mode=ro"
}

// sourceKind classifies a database file.
type sourceKind int

const (
	kindInvalid  sourceKind = iota
	kindSpace               // one space's database
	kindServer              // a server database
	kindCombined            // everything in one file: the single-file Go layout or a converted Laravel database
	kindLaravel             // a Laravel database not converted yet
)

// knownVersions are the migrations this build knows, for refusing files made
// by a newer app. "0001_initial" is the single-file layout's only version.
func knownVersions() map[string]bool {
	known := map[string]bool{"0001_initial": true}
	for _, set := range []migrate.Set{migrate.Server, migrate.Space} {
		names, _ := set.Versions()
		for _, n := range names {
			known[n] = true
		}
	}
	return known
}

// ErrNewerSchema is returned for a file made by a newer version of the app.
var ErrNewerSchema = errors.New("the file was made by a newer version of the app")

// inspect classifies the file at path without trusting it.
func inspect(ctx context.Context, path string) (sourceKind, error) {
	scratch, err := openScratch(ctx)
	if err != nil {
		return kindInvalid, err
	}
	defer scratch.Close()
	if err := attachSource(ctx, scratch, path); err != nil {
		return kindInvalid, err
	}
	tables, err := sourceTables(ctx, scratch)
	if err != nil {
		return kindInvalid, errNotSQLite
	}
	if tables["migrations"] && !tables["schema_migrations"] {
		return kindLaravel, nil
	}
	if tables["migrations"] {
		// Converted Laravel files keep their tracker; the stamp tells them apart.
		var stamp sql.NullString
		_ = scratch.QueryRowContext(ctx, `SELECT value FROM src.settings WHERE key = 'legacy_import_completed_at'`).Scan(&stamp)
		if !stamp.Valid {
			return kindLaravel, nil
		}
	}
	if tables["schema_migrations"] {
		rows, err := scratch.QueryContext(ctx, `SELECT version FROM src.schema_migrations`)
		if err != nil {
			return kindInvalid, errNotSQLite
		}
		known := knownVersions()
		for rows.Next() {
			var v string
			if rows.Scan(&v) == nil && !known[v] {
				rows.Close()
				return kindInvalid, ErrNewerSchema
			}
		}
		rows.Close()
	}
	switch {
	case tables["users"] && tables["accounts"]:
		return kindCombined, nil
	case tables["users"]:
		return kindServer, nil
	case tables["accounts"]:
		return kindSpace, nil
	}
	return kindInvalid, errNotSQLite
}

// openScratch opens an in-memory connection used to read a source file.
func openScratch(ctx context.Context) (*sql.DB, error) {
	scratch, err := sql.Open("sqlite", "file::memory:")
	if err != nil {
		return nil, err
	}
	scratch.SetMaxOpenConns(1)
	if _, err := scratch.ExecContext(ctx, `PRAGMA trusted_schema = OFF`); err != nil {
		_ = scratch.Close()
		return nil, err
	}
	return scratch, nil
}

// attachSource attaches path read-only as "src" and checks its integrity.
func attachSource(ctx context.Context, conn store.DB, path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `ATTACH DATABASE ? AS src`, srcURI(path)); err != nil {
		return errNotSQLite
	}
	var result string
	if err := conn.QueryRowContext(ctx, `PRAGMA src.quick_check`).Scan(&result); err != nil || result != "ok" {
		return errNotSQLite
	}
	return nil
}

// sourceTables lists the real tables of src (views and other objects are not
// tables and are never read).
func sourceTables(ctx context.Context, conn store.DB) (map[string]bool, error) {
	rows, err := conn.QueryContext(ctx, `SELECT name FROM src.sqlite_master WHERE type = 'table'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, rows.Err()
}

func tableColumns(ctx context.Context, conn store.DB, schema, table string) ([]string, error) {
	rows, err := conn.QueryContext(ctx, `SELECT name FROM pragma_table_info(?, ?)`, table, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func quoteIdent(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }

// sanitize creates dst with the migrations of plan.set and copies into it the
// rows of src that fit: tables of the set only, columns both have. Nothing of
// src's schema runs.
func sanitize(ctx context.Context, src, dst string, plan copyPlan) error {
	removeFiles(dst)
	fresh, err := db.Open(dst)
	if err != nil {
		return err
	}
	defer fresh.Close()
	if err := plan.set.Up(ctx, fresh); err != nil {
		return fmt.Errorf("prepare schema: %w", err)
	}
	if _, err := fresh.ExecContext(ctx, `PRAGMA trusted_schema = OFF`); err != nil {
		return err
	}
	if _, err := fresh.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	if err := attachSource(ctx, fresh, src); err != nil {
		return err
	}
	have, err := sourceTables(ctx, fresh)
	if err != nil {
		return errNotSQLite
	}
	rows, err := fresh.QueryContext(ctx, `SELECT name FROM main.sqlite_master
		WHERE type = 'table' AND name NOT IN ('schema_migrations', 'sqlite_sequence') ORDER BY name`)
	if err != nil {
		return err
	}
	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			rows.Close()
			return err
		}
		tables = append(tables, t)
	}
	rows.Close()

	tx, err := fresh.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, t := range tables {
		if !have[t] {
			continue
		}
		dstCols, err := tableColumns(ctx, tx, "main", t)
		if err != nil {
			return err
		}
		srcCols, err := tableColumns(ctx, tx, "src", t)
		if err != nil {
			return err
		}
		var cols, exprs []string
		for _, c := range dstCols {
			if !slices.Contains(srcCols, c) {
				continue
			}
			cols = append(cols, quoteIdent(c))
			if expr, ok := plan.transforms[t][c]; ok {
				exprs = append(exprs, expr)
			} else {
				exprs = append(exprs, quoteIdent(c))
			}
		}
		if len(cols) == 0 {
			continue
		}
		q := fmt.Sprintf(`INSERT INTO main.%s (%s) SELECT %s FROM src.%s`,
			quoteIdent(t), strings.Join(cols, ", "), strings.Join(exprs, ", "), quoteIdent(t))
		if _, err := tx.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("copy %s: %w", t, mapErr(err))
		}
		// Keep AUTOINCREMENT counters, so ids freed before the copy are never
		// handed out again.
		if have["sqlite_sequence"] {
			if err := copySequence(ctx, tx, t); err != nil {
				return err
			}
		}
	}
	if plan.set == migrate.Space && have["settings"] {
		if err := copySpaceSettings(ctx, tx); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return mapErr(err)
	}
	if _, err := fresh.ExecContext(ctx, `DETACH DATABASE src`); err != nil {
		return err
	}
	var violations int
	if err := fresh.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&violations); err != nil {
		return err
	}
	if violations > 0 {
		return fmt.Errorf("the file has %d rows pointing at rows that do not exist", violations)
	}
	_, err = fresh.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	return err
}

// copySequence raises table's AUTOINCREMENT counter in main to the source's.
// Copying the rows already set it to their highest id; a higher source value
// (rows deleted before the backup) must win so those ids stay retired.
func copySequence(ctx context.Context, tx *sql.Tx, table string) error {
	var seq sql.NullInt64
	err := tx.QueryRowContext(ctx, `SELECT seq FROM src.sqlite_sequence WHERE name = ?`, table).Scan(&seq)
	if errors.Is(err, sql.ErrNoRows) || !seq.Valid {
		return nil
	}
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE main.sqlite_sequence SET seq = MAX(seq, ?) WHERE name = ?`, seq.Int64, table)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		_, err = tx.ExecContext(ctx, `INSERT INTO main.sqlite_sequence (name, seq) VALUES (?, ?)`, table, seq.Int64)
	}
	return err
}

// spaceSettingKeys are the settings the single-file layout kept in settings
// and a space now keeps in space_settings.
var spaceSettingKeys = []string{"auto_update_currencies"}

func copySpaceSettings(ctx context.Context, tx *sql.Tx) error {
	for _, k := range spaceSettingKeys {
		if _, err := tx.ExecContext(ctx, `INSERT INTO main.space_settings (key, value)
			SELECT key, value FROM src.settings WHERE key = ?
			ON CONFLICT (key) DO NOTHING`, k); err != nil {
			return err
		}
	}
	return nil
}

// readSpaceUUID reads the space_uuid setting of a space database file.
func readSpaceUUID(ctx context.Context, path string) (string, error) {
	d, err := db.Open(path)
	if err != nil {
		return "", err
	}
	defer d.Close()
	var v sql.NullString
	err = d.QueryRowContext(ctx, `SELECT value FROM space_settings WHERE key = 'space_uuid'`).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) || !v.Valid {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	// Settings are stored JSON-encoded; a bare value is accepted too.
	return strings.Trim(v.String, `"`), nil
}

func writeSpaceUUID(ctx context.Context, path, uuid string) error {
	d, err := db.Open(path)
	if err != nil {
		return err
	}
	defer d.Close()
	_, err = d.ExecContext(ctx, `INSERT INTO space_settings (key, value) VALUES ('space_uuid', ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`, uuid)
	if err != nil {
		return err
	}
	_, err = d.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	return err
}

func fileSize(path string) int64 {
	var n int64
	for _, suffix := range []string{"", "-wal"} {
		if info, err := os.Stat(path + suffix); err == nil {
			n += info.Size()
		}
	}
	return n
}
