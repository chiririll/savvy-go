-- name: GetAccount :one
-- The same columns as ListAccounts, so its row converts to ListAccountsRow.
SELECT a.id, a.name, a.type, a.is_service, a.currency_id, a.initial_balance, a.is_active, a.sort_order, a.created_at,
	c.id AS currency_id_join, c.code, c.name AS currency_name, c.symbol, c.decimals, c.is_base, c.rate
FROM accounts a
JOIN currencies c ON c.id = a.currency_id
WHERE a.id = ?;

-- name: ListAccounts :many
-- exclude_service leaves out debt and space accounts: that is the list a picker
-- offers. exclude_space leaves out only the space accounts: net worth counts
-- debts but not the money that went to other spaces.
SELECT a.id, a.name, a.type, a.is_service, a.currency_id, a.initial_balance, a.is_active, a.sort_order, a.created_at,
	c.id AS currency_id_join, c.code, c.name AS currency_name, c.symbol, c.decimals, c.is_base, c.rate
FROM accounts a
JOIN currencies c ON c.id = a.currency_id
WHERE a.is_active = COALESCE(sqlc.narg('only_active'), a.is_active)
  AND CASE WHEN a.is_service THEN 1 ELSE 0 END
      <= CASE WHEN CAST(sqlc.narg('exclude_service') AS INTEGER) IS NULL THEN 1 ELSE 0 END
  AND CASE WHEN a.type = 'space' THEN 1 ELSE 0 END
      <= CASE WHEN CAST(sqlc.narg('exclude_space') AS INTEGER) IS NULL THEN 1 ELSE 0 END
ORDER BY CASE a.type WHEN 'debt' THEN 1 WHEN 'space' THEN 2 ELSE 0 END, a.sort_order, a.id;

-- name: NextAccountSortOrder :one
-- Debts are ordered among themselves, the other accounts among theirs.
SELECT COALESCE(MAX(sort_order), -1) + 1 FROM accounts
WHERE CASE WHEN sqlc.narg('debt_only') IS NULL THEN type IN ('bank','crypto','cash','credit') ELSE type = 'debt' END;

-- name: InsertAccount :execresult
INSERT INTO accounts (name, type, currency_id, initial_balance, is_active, sort_order, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateAccount :exec
UPDATE accounts SET name=?, type=?, currency_id=?, initial_balance=?, is_active=?, updated_at=?
WHERE id=?;

-- name: GetSpaceAccountID :one
-- The account of type space for a currency: at most one exists (unique index).
SELECT id FROM accounts WHERE type = 'space' AND currency_id = ?;

-- name: CountAccounts :one
SELECT COUNT(*) FROM accounts;

-- name: CountAccountPayments :one
SELECT COUNT(*) FROM payments WHERE account_id = ?;

-- name: DeleteAccount :exec
DELETE FROM accounts WHERE id = ?;

-- name: SetAccountSortOrder :exec
UPDATE accounts SET sort_order = ? WHERE id = ?;

-- name: AccountDailyDeltas :many
-- The net change of an account per day, oldest first, for confirmed
-- transactions up to as_of (all of them when NULL). A payment is signed money
-- in the account's own currency, so a change is just their sum; this is the
-- only place that reads how money moved. A balance is initial_balance plus
-- these rows, a balance history their running total.
-- CAST: sqlc would type a SUM as float; the sum of integers is an integer.
SELECT t.date AS day, CAST(SUM(p.amount) AS INTEGER) AS delta
FROM payments p
JOIN transactions t ON t.id = p.transaction_id
WHERE p.account_id = sqlc.arg('id') AND t.status = 'confirmed'
	AND t.date <= COALESCE(sqlc.narg('as_of'), t.date)
GROUP BY t.date
ORDER BY t.date;

-- name: GetAccountUnit :one
SELECT c.id, c.decimals FROM accounts a JOIN currencies c ON c.id = a.currency_id WHERE a.id = ?;

-- name: GetDebt :one
-- The same columns as ListDebts, so its row converts to ListDebtsRow.
SELECT a.id, a.name, a.type, a.is_service, a.currency_id, a.initial_balance, a.is_active, a.sort_order, a.created_at,
	d.direction, d.principal, d.due_date, d.counterparty, d.note, d.closed_at,
	c.id AS currency_id_join, c.code, c.name AS currency_name, c.symbol, c.decimals, c.is_base, c.rate
FROM debts d
JOIN accounts a ON a.id = d.account_id
JOIN currencies c ON c.id = a.currency_id
WHERE d.account_id = ?;

-- name: ListDebts :many
-- open_only leaves out closed debts.
SELECT a.id, a.name, a.type, a.is_service, a.currency_id, a.initial_balance, a.is_active, a.sort_order, a.created_at,
	d.direction, d.principal, d.due_date, d.counterparty, d.note, d.closed_at,
	c.id AS currency_id_join, c.code, c.name AS currency_name, c.symbol, c.decimals, c.is_base, c.rate
FROM debts d
JOIN accounts a ON a.id = d.account_id
JOIN currencies c ON c.id = a.currency_id
WHERE CASE WHEN d.closed_at IS NULL THEN 0 ELSE 1 END
      <= CASE WHEN CAST(sqlc.narg('open_only') AS INTEGER) IS NULL THEN 1 ELSE 0 END
ORDER BY a.sort_order, a.id;

-- name: InsertDebt :exec
INSERT INTO debts (account_id, direction, principal, due_date, counterparty, note)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateDebt :exec
UPDATE debts SET direction=?, principal=?, due_date=?, counterparty=?, note=? WHERE account_id=?;

-- name: CloseDebt :exec
UPDATE debts SET closed_at = ? WHERE account_id = ?;

-- name: ReopenDebt :exec
UPDATE debts SET closed_at = NULL WHERE account_id = ?;

-- name: CountDebtHistory :one
-- Payments and collections made on a debt; its issuance is not history.
SELECT COUNT(*) FROM payments p JOIN transactions t ON t.id = p.transaction_id
WHERE p.account_id = ? AND t.type IN ('debt_payment','debt_collection');

-- name: DeleteDebtIssuance :exec
-- The payments go with the transaction (cascade).
DELETE FROM transactions WHERE type IN ('debt_lend','debt_borrow')
	AND id IN (SELECT transaction_id FROM payments WHERE account_id = ?);

-- name: DeleteDebtAccount :exec
-- The debts row goes with the account (cascade).
DELETE FROM accounts WHERE id = ? AND type = 'debt';
