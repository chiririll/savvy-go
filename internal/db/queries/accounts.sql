-- name: ListAccounts :many
SELECT a.id, a.name, a.type, a.currency_id, a.initial_balance, a.is_active, a.sort_order,
	a.debt_type, a.target_amount, a.due_date, a.is_paid_off, a.counterparty, a.debt_description, a.created_at,
	c.id AS currency_id_join, c.code, c.name AS currency_name, c.symbol, c.decimals, c.is_base, c.rate
FROM accounts a
JOIN currencies c ON c.id = a.currency_id
WHERE a.id = COALESCE(sqlc.narg('id'), a.id)
  AND a.is_active = COALESCE(sqlc.narg('only_active'), a.is_active)
  AND CASE a.type WHEN 'bank' THEN 1 WHEN 'crypto' THEN 1 WHEN 'cash' THEN 1 ELSE 0 END
      >= COALESCE(CAST(sqlc.narg('exclude_debts') AS INTEGER), 0)
  AND CASE a.type WHEN 'debt' THEN 1 ELSE 0 END >= COALESCE(CAST(sqlc.narg('only_debts') AS INTEGER), 0)
  AND a.is_paid_off <= CASE WHEN CAST(sqlc.narg('unpaid_only') AS INTEGER) IS NULL THEN 1 ELSE 0 END
ORDER BY CASE WHEN a.type = 'debt' THEN 1 ELSE 0 END, a.sort_order, a.id;

-- name: NextAccountSortOrder :one
SELECT COALESCE(MAX(sort_order), -1) + 1 FROM accounts
WHERE CASE WHEN sqlc.narg('debt_only') IS NULL THEN type IN ('bank','crypto','cash') ELSE type = 'debt' END;

-- name: InsertAccount :execresult
INSERT INTO accounts (name, type, debt_type, currency_id, initial_balance, target_amount, due_date,
	is_paid_off, counterparty, debt_description, is_active, sort_order, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateAccount :exec
UPDATE accounts SET name=?, type=?, currency_id=?, initial_balance=?, is_active=?,
	debt_type=?, target_amount=?, due_date=?, is_paid_off=?, counterparty=?, debt_description=?, updated_at=?
WHERE id=?;

-- name: CountAccounts :one
SELECT COUNT(*) FROM accounts;

-- name: CountAccountTransactions :one
SELECT COUNT(*) FROM transactions WHERE account_id = ? OR to_account_id = ?;

-- name: DeleteAccount :exec
DELETE FROM accounts WHERE id = ?;

-- name: SetAccountSortOrder :exec
UPDATE accounts SET sort_order = ? WHERE id = ?;

-- name: SumDebtPayments :one
SELECT CAST(COALESCE(SUM(to_amount),0) AS INTEGER) FROM transactions
WHERE to_account_id = ? AND status = 'confirmed' AND type IN ('debt_payment','debt_collection');

-- name: AccountDailyDeltas :many
-- The net change of a non-debt account per day, oldest first, for confirmed
-- transactions up to as_of (all of them when NULL). This is the only place that
-- knows how a transaction moves money: it adds to the account for income,
-- debt_collection and debt_borrow, takes from it for every other type, and a
-- transfer adds to_amount to its destination account (a transfer from an account
-- to itself therefore does both). A balance is the sum of these rows, a balance
-- history their running total.
-- CAST: sqlc would type a SUM as float; the sum of integers is an integer.
WITH p AS (SELECT CAST(sqlc.arg('id') AS INTEGER) AS id)
SELECT t.date AS day,
	CAST(SUM(
		CASE WHEN t.account_id = p.id THEN
			CASE WHEN t.type IN ('income', 'debt_collection', 'debt_borrow') THEN t.amount ELSE -t.amount END
		ELSE 0 END
		+ CASE WHEN t.to_account_id = p.id AND t.type = 'transfer' THEN COALESCE(t.to_amount, 0) ELSE 0 END
	) AS INTEGER) AS delta
FROM transactions t, p
WHERE t.status = 'confirmed' AND (t.account_id = p.id OR t.to_account_id = p.id)
	AND t.date <= COALESCE(sqlc.narg('as_of'), t.date)
GROUP BY t.date
ORDER BY t.date;

-- name: GetAccountUnit :one
SELECT c.id, c.decimals FROM accounts a JOIN currencies c ON c.id = a.currency_id WHERE a.id = ?;

-- name: MarkAccountPaidOff :exec
UPDATE accounts SET is_paid_off = 1 WHERE id = ?;

-- name: ReopenAccount :exec
UPDATE accounts SET is_paid_off = 0, updated_at = ? WHERE id = ?;

-- name: CountDebtHistory :one
SELECT COUNT(*) FROM transactions
WHERE to_account_id = ? AND type IN ('debt_payment','debt_collection');

-- name: DeleteDebtIssuance :exec
DELETE FROM transactions WHERE to_account_id = ? AND type IN ('debt_lend','debt_borrow');

-- name: DeleteDebtAccount :exec
DELETE FROM accounts WHERE id = ? AND type = 'debt';
