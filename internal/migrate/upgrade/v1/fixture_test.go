package v1_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"savvy-go/internal/db"
	v1 "savvy-go/internal/migrate/upgrade/v1"
)

// fixtureSQL is a v1.0 space with one row of every kind the conversion maps
// (docs/plans/transactions-erd.md, "v1.0 migration"). The ids are the ones
// the expectations name.
//
// Accounts: 1 card EUR, 2 cash RSD, 3 a loan I made, 4 a mortgage I owe,
// 5 a debt recorded as it stood (no issuance), closed.
const fixtureSQL = `
INSERT INTO currencies (id, code, name, symbol, decimals, is_base, rate) VALUES
	(1, 'EUR', 'Euro', '€', 2, 1, '1'), (2, 'RSD', 'Dinar', 'din', 2, 0, '117'), (3, 'JPY', 'Yen', '¥', 0, 0, '160');
INSERT INTO categories (id, name, type) VALUES (1, 'Food', 'expense'), (2, 'Salary', 'income');
INSERT INTO tags (id, name) VALUES (1, 'home'), (2, 'work');
INSERT INTO accounts (id, name, type, debt_type, currency_id, initial_balance, target_amount, due_date, is_paid_off,
		counterparty, debt_description, sort_order, updated_at) VALUES
	(1, 'Card', 'bank', NULL, 1, 10000, NULL, NULL, 0, NULL, NULL, 0, NULL),
	(2, 'Cash', 'cash', NULL, 2, 0, NULL, NULL, 0, NULL, NULL, 1, NULL),
	(3, 'Loan to Sam', 'debt', 'owed_to_me', 1, 0, 50000, '2027-01-01', 0, 'Sam', 'for the car', 0, NULL),
	(4, 'Mortgage', 'debt', 'i_owe', 1, 0, 100000, NULL, 0, 'Bank', NULL, 1, NULL),
	(5, 'Old IOU', 'debt', 'owed_to_me', 1, 0, 3000, NULL, 1, 'Ann', NULL, 2, '2026-02-01T00:00:00Z');

INSERT INTO space_transfers (uuid, from_space_uuid, from_account_id, from_amount, from_currency, from_decimals,
		to_space_uuid, to_account_id, to_amount, to_currency, to_decimals, date, description, created_by, version,
		deleted_at, sig_v, signer_kid, signature, status, review, remote) VALUES
	('u1', 'sp-a', 1, 4000, 'EUR', 2, 'sp-b', 9, 468000, 'RSD', 2, '2026-02-10', 'Rent <share> & co', 1, 1,
		NULL, 1, 'k1', 'sig1', 'ok', NULL, NULL),
	('u2', 'sp-b', 9, 468000, 'RSD', 2, 'sp-a', 2, 117000, 'RSD', 2, '2026-02-11', NULL, NULL, 1,
		NULL, 1, 'k1', 'sig2', 'needs_attention', NULL, NULL),
	('u3', 'sp-b', 9, 100, 'EUR', 2, 'sp-a', 1, 100, 'EUR', 2, '2026-02-12', NULL, NULL, 2,
		NULL, 1, 'k2', 'sig3', 'pending', 'created_remote', NULL),
	('u4', 'sp-a', 1, 700, 'EUR', 2, 'sp-b', 9, 700, 'EUR', 2, '2026-02-13', 'Fee', 2, 3,
		NULL, 1, 'k1', 'sig4', 'pending', 'changed_remote', '{"uuid":"u4","version":4}');

INSERT INTO recurring_transactions (id, type, account_id, to_account_id, category_id, amount, to_amount, is_estimated,
		description, frequency, interval, day_of_week, day_of_month, start_date, end_date, next_run_date,
		last_run_date, is_active, created_at) VALUES
	(1, 'expense', 1, NULL, 1, 1500, NULL, 1, 'Groceries', 'monthly', 1, NULL, 1, '2026-01-01', NULL,
		'2026-03-01', '2026-02-01', 1, '2026-01-01T00:00:00Z'),
	(2, 'transfer', 1, 2, NULL, 1000, NULL, 0, NULL, 'weekly', 2, 1, 15, '2026-01-05', '2026-12-31',
		'2026-02-02', NULL, 0, '2026-01-01T00:00:00Z'),
	(3, 'income', 2, NULL, 2, 0, NULL, 1, NULL, 'daily', 1, NULL, NULL, '2026-01-01', NULL,
		'2026-01-01', NULL, 1, '2026-01-01T00:00:00Z');
INSERT INTO recurring_transaction_tag (recurring_transaction_id, tag_id) VALUES (1, 2), (1, 1);

INSERT INTO transactions (id, type, account_id, to_account_id, category_id, amount, to_amount, is_estimated,
		description, dedup_hash, date, status, recurring_transaction_id, space_transfer_uuid, created_by, created_at) VALUES
	(1, 'expense', 1, NULL, 1, 1250, NULL, 0, 'Shop', 'abc', '2026-02-01', 'confirmed', NULL, NULL, 1, '2026-02-01T10:00:00Z'),
	(2, 'income', 1, NULL, 2, 300000, NULL, 0, 'Pay', NULL, '2026-02-02', 'confirmed', NULL, NULL, 1, '2026-02-02T10:00:00Z'),
	(3, 'transfer', 1, 2, NULL, 1000, 117000, 0, NULL, NULL, '2026-02-03', 'confirmed', NULL, NULL, 1, NULL),
	(4, 'transfer', 1, 1, NULL, 500, 400, 0, 'Round trip', NULL, '2026-02-04', 'confirmed', NULL, NULL, 1, NULL),
	(5, 'expense', 1, NULL, 1, 0, NULL, 0, 'Free', NULL, '2026-02-05', 'confirmed', NULL, NULL, 1, NULL),
	(6, 'debt_lend', 1, 3, NULL, 50000, NULL, 0, NULL, NULL, '2026-02-06', 'confirmed', NULL, NULL, 1, NULL),
	(7, 'debt_collection', 1, 3, NULL, 2000, NULL, 0, NULL, NULL, '2026-02-07', 'confirmed', NULL, NULL, 1, NULL),
	(8, 'debt_borrow', 1, 4, NULL, 100000, NULL, 0, NULL, NULL, '2026-02-08', 'confirmed', NULL, NULL, 1, NULL),
	(9, 'debt_payment', 1, 4, NULL, 5000, NULL, 0, NULL, NULL, '2026-02-09', 'confirmed', NULL, NULL, 1, NULL),
	(10, 'transfer_out', 1, NULL, NULL, 4000, NULL, 0, 'Rent <share> & co', NULL, '2026-02-10', 'confirmed', NULL, 'u1', 1, NULL),
	(11, 'transfer_in', 2, NULL, NULL, 117000, NULL, 0, NULL, NULL, '2026-02-11', 'confirmed', NULL, 'u2', 1, NULL),
	(12, 'expense', 1, NULL, 1, 1500, NULL, 1, 'Groceries', NULL, '2026-03-03', 'pending', 1, NULL, NULL, NULL),
	(13, 'expense', 1, NULL, 1, 1400, NULL, 0, 'Groceries', NULL, '2026-02-01', 'confirmed', 1, NULL, NULL, NULL),
	(14, 'expense', 1, NULL, 1, 90, NULL, 0, NULL, NULL, '2026-02-14', 'skipped', NULL, NULL, 1, NULL);
INSERT INTO transaction_items (id, transaction_id, name, quantity, price_per_unit, total_price) VALUES
	(1, 1, ' Milk ', '2', 300, 600), (2, 1, 'Bread', '1', 650, 650), (3, 14, '  ', '1', 90, 90);
INSERT INTO transaction_tag (transaction_id, tag_id) VALUES (1, 1), (12, 2);
INSERT INTO transactions (id, type, account_id, amount, date, status) VALUES (20, 'expense', 1, 1, '2026-02-20', 'confirmed');
DELETE FROM transactions WHERE id = 20;
`

// openV1 makes a v1.0 space file with fixtureSQL and returns its path.
func openV1(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "space.sqlite")
	d, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	ctx := context.Background()
	for _, stmt := range []string{
		`CREATE TABLE schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL) STRICT`,
		`INSERT INTO schema_migrations VALUES ('space/0001_initial', '2026-01-01T00:00:00Z')`,
	} {
		if _, err := d.ExecContext(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}
	if err := execAll(ctx, d, v1.Schema); err != nil {
		t.Fatal(err)
	}
	if err := execAll(ctx, d, fixtureSQL); err != nil {
		t.Fatal(err)
	}
	return path
}

// execAll runs a script of statements, each ending a line with a semicolon.
func execAll(ctx context.Context, d *sql.DB, script string) error {
	for _, stmt := range v1.Statements(script) {
		if _, err := d.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
