package v1

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const migrationsTable = `CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TEXT NOT NULL
) STRICT`

// Apply makes db a v1.0 space file: it creates the frozen schema and records
// Version, unless that is recorded already. Importing a Laravel database goes
// through this shape (internal/legacy), and so does reading an untrusted v1.0
// file, before Convert turns either into the current schema.
func Apply(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, migrationsTable); err != nil {
		return err
	}
	var done int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, Version).Scan(&done); err != nil {
		return err
	}
	if done > 0 {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, stmt := range Statements(Schema) {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply %s: %s: %w", Version, firstLine(stmt), err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
		Version, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return err
	}
	return tx.Commit()
}

// EnsureIndexes creates the indexes of the v1.0 schema that are missing.
func EnsureIndexes(ctx context.Context, db *sql.DB) error {
	for _, stmt := range Statements(Schema) {
		up := strings.ToUpper(strings.TrimSpace(stmt))
		if !strings.HasPrefix(up, "CREATE INDEX") && !strings.HasPrefix(up, "CREATE UNIQUE INDEX") {
			continue
		}
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("%s: %w", firstLine(stmt), err)
		}
	}
	return nil
}
