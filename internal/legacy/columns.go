package legacy

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

type addColumn struct {
	Name string
	DDL  string // type + constraints, e.g. "TEXT" or "INTEGER NOT NULL DEFAULT 0"
}

// Extra columns that later Laravel migrations added. Applied to an existing
// Laravel-era table before Go CREATE INDEX statements run.
var extraColumns = map[string][]addColumn{
	"users": {
		{"role", "TEXT NOT NULL DEFAULT 'admin'"},
		{"is_sso_only", "INTEGER NOT NULL DEFAULT 0"},
		{"two_factor_secret", "TEXT"},
		{"two_factor_enabled", "INTEGER NOT NULL DEFAULT 0"},
		{"two_factor_confirmed", "INTEGER NOT NULL DEFAULT 0"},
	},
	"auth_sessions": {
		{"remember_me", "INTEGER NOT NULL DEFAULT 0"},
		{"refreshed_at", "TEXT"},
	},
	"currencies": {
		{"is_base", "INTEGER NOT NULL DEFAULT 0"},
		{"rate", "REAL NOT NULL DEFAULT 1"},
	},
	"accounts": {
		{"debt_type", "TEXT"},
		{"target_amount", "REAL"},
		{"due_date", "TEXT"},
		{"is_paid_off", "INTEGER NOT NULL DEFAULT 0"},
		{"counterparty", "TEXT"},
		{"debt_description", "TEXT"},
		{"sort_order", "INTEGER NOT NULL DEFAULT 0"},
	},
	"transactions": {
		{"dedup_hash", "TEXT"},
		{"status", "TEXT NOT NULL DEFAULT 'confirmed'"},
		{"recurring_transaction_id", "INTEGER"},
	},
	"budgets": {
		{"currency_id", "INTEGER"},
	},
	"backups": {
		{"app_version", "TEXT"},
		{"schema_migrations", "TEXT"},
	},
}

// EnsureColumns adds missing domain columns on a Laravel-era database so the
// Go schema indexes can be created. No-op when the file is not Laravel.
func EnsureColumns(ctx context.Context, db *sql.DB) error {
	if !IsLaravel(ctx, db) {
		return nil
	}
	for table, cols := range extraColumns {
		if !tableExists(ctx, db, table) {
			continue
		}
		have, err := columns(ctx, db, table)
		if err != nil {
			return err
		}
		set := make(map[string]bool, len(have))
		for _, c := range have {
			set[c] = true
		}
		for _, col := range cols {
			if set[col.Name] {
				continue
			}
			ddl := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, col.Name, col.DDL)
			if _, err := db.ExecContext(ctx, ddl); err != nil {
				return fmt.Errorf("%s: %w", ddl, err)
			}
			slog.Info("legacy added column", "table", table, "column", col.Name)
		}
	}
	return nil
}
