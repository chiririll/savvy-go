package v1

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"savvy-go/internal/db"
	"savvy-go/internal/migrate"
)

// IsV1 reports whether the space file at path is a v1.0 one: it records
// space/0001_initial and not the current base. A file that was never migrated
// is not.
func IsV1(ctx context.Context, path string) (bool, error) {
	d, err := db.OpenReadOnly(path)
	if err != nil {
		return false, err
	}
	defer d.Close()
	var table string
	err = d.QueryRowContext(ctx,
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations'`).Scan(&table)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var initial, base int
	err = d.QueryRowContext(ctx, `SELECT
		COALESCE(SUM(version = ?), 0), COALESCE(SUM(version = ?), 0) FROM schema_migrations`,
		Version, migrate.BaseVersion).Scan(&initial, &base)
	return initial > 0 && base == 0, err
}

// Convert writes the space base and the data of the v1.0 file src into a new
// file dst, then applies the migrations that came after the base. The rows are
// copied into the base, not into the latest schema, so a later migration can
// rename or drop what the copy writes. src is only read. dst must not exist;
// when the conversion fails nothing is left at dst.
func Convert(ctx context.Context, src, dst string) (err error) {
	if _, statErr := os.Stat(dst); statErr == nil {
		return fmt.Errorf("convert v1.0 space: %s already exists", dst)
	}
	fresh, err := db.Open(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = fresh.Close()
		if err != nil {
			db.RemoveFiles(dst)
		}
	}()
	if err = migrate.Space.UpTo(ctx, fresh, migrate.BaseVersion); err != nil {
		return fmt.Errorf("prepare schema: %w", err)
	}
	for _, pragma := range []string{`PRAGMA trusted_schema = OFF`, `PRAGMA foreign_keys = OFF`} {
		if _, err = fresh.ExecContext(ctx, pragma); err != nil {
			return err
		}
	}
	if _, err = fresh.ExecContext(ctx, `ATTACH DATABASE ? AS src`, db.ReadOnlyURI(src)); err != nil {
		return fmt.Errorf("open the v1.0 file: %w", err)
	}
	if err = copyAll(ctx, fresh); err != nil {
		return err
	}
	if _, err = fresh.ExecContext(ctx, `DETACH DATABASE src`); err != nil {
		return err
	}
	var violations int
	if err = fresh.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&violations); err != nil {
		return err
	}
	if violations > 0 {
		return fmt.Errorf("the converted file has %d rows pointing at rows that do not exist", violations)
	}
	if err = migrate.Space.Up(ctx, fresh); err != nil {
		return fmt.Errorf("migrate after the base: %w", err)
	}
	return db.Checkpoint(ctx, fresh)
}

// srcURI is the read-only URI of a file for ATTACH.
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

// Statements splits a script into its statements: a statement ends with a
// line that ends with a semicolon, and lines that are comments are dropped.
func Statements(script string) []string {
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
			if stmt := strings.TrimSuffix(strings.TrimSpace(current.String()), ";"); stmt != "" {
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

// copyAll copies every row in one transaction.
func copyAll(ctx context.Context, fresh *sql.DB) error {
	tx, err := fresh.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	steps := []struct {
		name string
		run  func(context.Context, *sql.Tx) error
	}{
		{"shared tables", copyShared},
		{"accounts and debts", copyAccounts},
		{"transactions", copyTransactions},
		{"space transfers", copyTransfers},
		{"recurring rules", copyRules},
	}
	for _, s := range steps {
		if err := s.run(ctx, tx); err != nil {
			return fmt.Errorf("convert %s: %w", s.name, err)
		}
	}
	return tx.Commit()
}
