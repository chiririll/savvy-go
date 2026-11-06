package v1

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// shared are the tables the new base keeps as they were. Their rows are
// copied column by column, for the columns both files have.
var shared = []string{
	"currencies", "categories", "tags", "budgets", "budget_category", "budget_tag",
	"automation_rules", "automation_rule_logs", "transaction_imports", "space_settings",
}

func copyShared(ctx context.Context, tx *sql.Tx) error {
	for _, table := range shared {
		if err := copyTable(ctx, tx, table); err != nil {
			return fmt.Errorf("%s: %w", table, err)
		}
	}
	return nil
}

// copyTable copies the columns table has in both files; a table the v1.0
// file lacks stays empty.
func copyTable(ctx context.Context, tx *sql.Tx, table string) error {
	cols, err := sharedColumns(ctx, tx, table)
	if err != nil || len(cols) == 0 {
		return err
	}
	list := strings.Join(cols, ", ")
	if _, err := tx.ExecContext(ctx,
		fmt.Sprintf(`INSERT INTO main.%s (%s) SELECT %s FROM src.%s`, table, list, list, table)); err != nil {
		return err
	}
	return bumpSequence(ctx, tx, table)
}

func sharedColumns(ctx context.Context, tx *sql.Tx, table string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT m.name FROM pragma_table_info(?, 'main') m
		WHERE m.name IN (SELECT s.name FROM pragma_table_info(?, 'src') s) ORDER BY m.cid`, table, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		cols = append(cols, `"`+c+`"`)
	}
	return cols, rows.Err()
}

// bumpSequence makes the AUTOINCREMENT counter of table at least what the
// v1.0 file had, so an id that was handed out once is not handed out again.
func bumpSequence(ctx context.Context, tx *sql.Tx, table string) error {
	for _, stmt := range []string{
		`INSERT INTO main.sqlite_sequence (name, seq) SELECT ?1, 0
			WHERE NOT EXISTS (SELECT 1 FROM main.sqlite_sequence WHERE name = ?1)`,
		`UPDATE main.sqlite_sequence SET seq = MAX(seq, COALESCE(
			(SELECT seq FROM src.sqlite_sequence WHERE name = ?1), 0)) WHERE name = ?1`,
	} {
		if _, err := tx.ExecContext(ctx, stmt, table); err != nil {
			return err
		}
	}
	return nil
}

// debtBasis is what a debt account's balance is rebuilt from: its principal
// (the target, else what was issued, else 1 as the base wants it above 0) and
// what was issued. The balance was target - paid; it is now initial_balance
// plus the payments on the account, so initial_balance = +-(principal - issued).
const debtBasis = `CREATE TEMP TABLE debt_basis AS
	SELECT a.id AS id,
		COALESCE(NULLIF(a.target_amount, 0), NULLIF(i.total, 0), 1) AS principal,
		COALESCE(i.total, 0) AS issued
	FROM src.accounts a
	LEFT JOIN (SELECT to_account_id AS id, SUM(COALESCE(to_amount, amount)) AS total FROM src.transactions
		WHERE type IN ('debt_lend', 'debt_borrow') GROUP BY to_account_id) i ON i.id = a.id
	WHERE a.type = 'debt'`

var accountSteps = []string{
	debtBasis,
	`INSERT INTO main.accounts (id, name, type, currency_id, initial_balance, is_active, sort_order, created_at, updated_at)
	SELECT a.id, a.name, a.type, a.currency_id,
		CASE WHEN a.type = 'debt'
			THEN (CASE COALESCE(a.debt_type, 'owed_to_me') WHEN 'owed_to_me' THEN 1 ELSE -1 END) * (b.principal - b.issued)
			ELSE a.initial_balance END,
		a.is_active, a.sort_order, a.created_at, a.updated_at
	FROM src.accounts a LEFT JOIN debt_basis b ON b.id = a.id`,
	`INSERT INTO main.debts (account_id, direction, principal, due_date, counterparty, note, closed_at)
	SELECT a.id, COALESCE(a.debt_type, 'owed_to_me'), b.principal, a.due_date, a.counterparty, a.debt_description,
		CASE WHEN a.is_paid_off = 1 THEN COALESCE(a.updated_at, strftime('%Y-%m-%dT%H:%M:%SZ', 'now')) END
	FROM src.accounts a JOIN debt_basis b ON b.id = a.id`,
	`DROP TABLE debt_basis`,
}

// spaceAccounts makes the account of type space for each currency a transfer
// to or from another space was made in.
const spaceAccounts = `INSERT INTO main.accounts (name, type, currency_id, initial_balance, is_active, sort_order, created_at, updated_at)
	SELECT 'Other spaces · ' || c.code, 'space', c.id, 0, 1, 0,
		strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
	FROM main.currencies c
	WHERE c.id IN (SELECT a.currency_id FROM src.transactions t JOIN src.accounts a ON a.id = t.account_id
		WHERE t.type IN ('transfer_in', 'transfer_out'))
	ORDER BY c.id`

func copyAccounts(ctx context.Context, tx *sql.Tx) error {
	for _, stmt := range accountSteps {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	if err := bumpSequence(ctx, tx, "accounts"); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, spaceAccounts)
	return err
}

// transactionSteps follow the mapping table of the plan, row kind by row kind.
var transactionSteps = []string{
	`INSERT INTO main.transactions (id, type, description, date, status, external_id, created_by, created_at, updated_at)
	SELECT id, type, description, date, status,
		CASE WHEN dedup_hash IS NOT NULL THEN 'csv:' || account_id || ':' || dedup_hash END,
		created_by, created_at, updated_at
	FROM src.transactions`,

	// income and expense: one payment on the account
	`INSERT INTO main.payments (transaction_id, account_id, amount)
	SELECT id, account_id, CASE type WHEN 'income' THEN amount ELSE -amount END
	FROM src.transactions WHERE type IN ('income', 'expense')`,

	// transfer: out of one account, into the other (a transfer to itself: the net)
	`INSERT INTO main.payments (transaction_id, account_id, amount)
	SELECT id, account_id,
		CASE WHEN to_account_id = account_id THEN COALESCE(to_amount, amount) - amount ELSE -amount END
	FROM src.transactions WHERE type = 'transfer'`,
	`INSERT INTO main.payments (transaction_id, account_id, amount)
	SELECT id, to_account_id, COALESCE(to_amount, amount)
	FROM src.transactions
	WHERE type = 'transfer' AND to_account_id IS NOT NULL AND to_account_id <> account_id`,

	// transfer_in and transfer_out: the account, and the space account that carries the uuid
	`INSERT INTO main.payments (transaction_id, account_id, amount)
	SELECT id, account_id, CASE type WHEN 'transfer_in' THEN amount ELSE -amount END
	FROM src.transactions WHERE type IN ('transfer_in', 'transfer_out')`,
	`INSERT INTO main.payments (transaction_id, account_id, amount, space_transfer_uuid)
	SELECT t.id, s.id, CASE t.type WHEN 'transfer_in' THEN -t.amount ELSE t.amount END, t.space_transfer_uuid
	FROM src.transactions t
	JOIN src.accounts a ON a.id = t.account_id
	JOIN main.accounts s ON s.type = 'space' AND s.currency_id = a.currency_id
	WHERE t.type IN ('transfer_in', 'transfer_out')`,

	// debts: the ordinary account as AccountDailyDeltas moved it, the debt account with the debt sign
	`INSERT INTO main.payments (transaction_id, account_id, amount)
	SELECT id, account_id, CASE WHEN type IN ('debt_collection', 'debt_borrow') THEN amount ELSE -amount END
	FROM src.transactions WHERE type IN ('debt_lend', 'debt_borrow', 'debt_payment', 'debt_collection')`,
	`INSERT INTO main.payments (transaction_id, account_id, amount)
	SELECT id, to_account_id,
		CASE WHEN type IN ('debt_lend', 'debt_payment') THEN COALESCE(to_amount, amount) ELSE -COALESCE(to_amount, amount) END
	FROM src.transactions
	WHERE type IN ('debt_lend', 'debt_borrow', 'debt_payment', 'debt_collection')
		AND to_account_id IS NOT NULL AND to_account_id <> account_id`,

	// income and expense of more than 0: one line, noted with the item names
	`INSERT INTO main.transaction_lines (transaction_id, seq, category_id, currency_id, amount, note)
	SELECT t.id, 0, t.category_id, a.currency_id, t.amount,
		(SELECT group_concat(trim(i.name), ', ' ORDER BY i.id) FROM src.transaction_items i
			WHERE i.transaction_id = t.id AND trim(i.name) <> '')
	FROM src.transactions t JOIN src.accounts a ON a.id = t.account_id
	WHERE t.type IN ('income', 'expense') AND t.amount > 0`,

	`INSERT INTO main.transaction_tag (transaction_id, tag_id) SELECT transaction_id, tag_id FROM src.transaction_tag`,

	// the handled due dates of the rules; a pending occurrence is due on the rule's next run
	`INSERT OR IGNORE INTO main.recurring_instances (recurring_transaction_id, due_date, transaction_id)
	SELECT t.recurring_transaction_id,
		CASE WHEN t.status = 'pending' THEN r.next_run_date ELSE COALESCE(t.date, r.start_date) END, t.id
	FROM src.transactions t JOIN src.recurring_transactions r ON r.id = t.recurring_transaction_id
	ORDER BY t.status = 'pending' DESC, t.id`,
}

func copyTransactions(ctx context.Context, tx *sql.Tx) error {
	for _, stmt := range transactionSteps {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("%s: %w", firstLine(stmt), err)
		}
	}
	return bumpSequence(ctx, tx, "transactions")
}

func firstLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 70 {
		s = s[:70] + "…"
	}
	return s
}
