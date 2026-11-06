package v1_test

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"savvy-go/internal/db"
	v1 "savvy-go/internal/migrate/upgrade/v1"
)

// v1.0 had no credit card: a card was a debt with no target that money was
// spent from. It becomes an ordinary account (the user can make it a credit
// card), its payments as they were, and its debt payments become transfers.
// A debt with a target or an issuance stays a debt, whatever was spent from it.
const cardDebtSQL = `
INSERT INTO currencies (id, code, name, symbol, decimals, is_base, rate) VALUES (1, 'USD', 'Dollar', '$', 2, 1, '1');
INSERT INTO categories (id, name, type) VALUES (1, 'Food', 'expense');
INSERT INTO accounts (id, name, type, debt_type, currency_id, initial_balance, target_amount, counterparty,
		debt_description, sort_order) VALUES
	(1, 'Checking', 'bank', NULL, 1, 100000, NULL, NULL, NULL, 0),
	(2, 'Amex', 'debt', 'i_owe', 1, 0, 0, 'American Express', 'Credit card balance', 0),
	(3, 'Mortgage', 'debt', 'i_owe', 1, 0, 100000, 'Bank', NULL, 1),
	(4, 'Loan', 'debt', 'i_owe', 1, 0, 0, 'Sam', NULL, 2),
	(5, 'Friend', 'debt', 'owed_to_me', 1, 0, 0, 'Ann', NULL, 3);
INSERT INTO transactions (id, type, account_id, to_account_id, category_id, amount, date, status) VALUES
	(1, 'expense', 2, NULL, 1, 3000, '2026-02-01', 'confirmed'),
	(2, 'expense', 2, NULL, 1, 2000, '2026-02-02', 'confirmed'),
	(3, 'debt_payment', 1, 2, NULL, 4000, '2026-02-10', 'confirmed'),
	(4, 'debt_borrow', 1, 3, NULL, 100000, '2026-02-03', 'confirmed'),
	(5, 'expense', 3, NULL, 1, 700, '2026-02-04', 'confirmed'),
	(6, 'debt_borrow', 1, 4, NULL, 800, '2026-02-05', 'confirmed'),
	(7, 'expense', 4, NULL, 1, 100, '2026-02-06', 'confirmed'),
	(8, 'debt_lend', 1, 5, NULL, 900, '2026-02-07', 'confirmed');
`

func TestCardDebtBecomesOrdinaryAccount(t *testing.T) {
	ctx := context.Background()
	src := filepath.Join(t.TempDir(), "space.sqlite")
	d, err := db.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`CREATE TABLE schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL) STRICT`,
		`INSERT INTO schema_migrations VALUES ('space/0001_initial', '2026-01-01T00:00:00Z')`,
	} {
		if _, err := d.ExecContext(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}
	for _, script := range []string{v1.Schema, cardDebtSQL} {
		if err := execAll(ctx, d, script); err != nil {
			t.Fatal(err)
		}
	}
	d.Close()

	dst := filepath.Join(t.TempDir(), "new.sqlite")
	if err := v1.Convert(ctx, src, dst); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	out, err := db.Open(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	got := rowsOf(t, out, `SELECT a.id, a.name, a.type, a.initial_balance, d.account_id IS NOT NULL
		FROM accounts a LEFT JOIN debts d ON d.account_id = a.id WHERE a.type <> 'space' ORDER BY a.id`)
	wantAccounts := []string{
		"1|Checking|bank|100000|0",
		"2|Amex|bank|0|0",     // a card: an ordinary account that starts at zero, no debts row
		"3|Mortgage|debt|0|1", // a target
		"4|Loan|debt|0|1",     // no target, but money was issued to it: a debt, not a card
		"5|Friend|debt|0|1",   // owed to me: never a card
	}
	if !reflect.DeepEqual(got, wantAccounts) {
		t.Fatalf("accounts = %v, want %v", got, wantAccounts)
	}

	// The card's balance is what it was: spent 5000, paid 4000.
	bal := rowsOf(t, out, `SELECT COALESCE(SUM(p.amount), 0) FROM payments p WHERE p.account_id = 2`)
	if !reflect.DeepEqual(bal, []string{"-1000"}) {
		t.Errorf("card payments sum = %v, want -1000", bal)
	}
	types := rowsOf(t, out, `SELECT id, type FROM transactions WHERE id IN (3, 4, 6, 8) ORDER BY id`)
	if !reflect.DeepEqual(types, []string{"3|transfer", "4|debt_borrow", "6|debt_borrow", "8|debt_lend"}) {
		t.Errorf("transaction types = %v", types)
	}
	// The payment to the card moved money out of one account and into the other.
	pays := rowsOf(t, out, `SELECT account_id, amount FROM payments WHERE transaction_id = 3 ORDER BY account_id`)
	if !reflect.DeepEqual(pays, []string{"1|-4000", "2|4000"}) {
		t.Errorf("payments of the card payment = %v", pays)
	}
}
