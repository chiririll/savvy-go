-- name: InsertTransaction :execresult
INSERT INTO transactions (type, account_id, to_account_id, category_id, amount, to_amount, is_estimated,
	description, date, status, recurring_transaction_id, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?);

-- name: UpdateTransaction :exec
UPDATE transactions SET type=?, account_id=?, to_account_id=?, category_id=?, amount=?, to_amount=?,
	is_estimated=?, description=?, date=?, updated_at=? WHERE id=?;

-- name: DeleteTransactionItems :exec
DELETE FROM transaction_items WHERE transaction_id = ?;

-- name: DeleteTransaction :exec
DELETE FROM transactions WHERE id = ?;

-- name: ConfirmTransaction :exec
-- A confirmed transaction has its real amount, so it is no longer an estimate.
UPDATE transactions SET status='confirmed', amount=?, to_amount=?, is_estimated=0, date=?, updated_at=? WHERE id=?;

-- name: SkipTransaction :exec
UPDATE transactions SET status='skipped', updated_at=? WHERE id=?;

-- name: CountAllTransactions :one
SELECT COUNT(*) FROM transactions;

-- name: GetTransaction :one
-- Lists go through filter.ListTransactions, which selects the same columns.
SELECT t.id, t.type, t.account_id, t.to_account_id, t.category_id, t.amount, t.to_amount,
	t.is_estimated, t.description, t.date, t.status, t.recurring_transaction_id, t.created_at,
	ca.id AS currency_id, ca.decimals AS decimals,
	COALESCE(cb.id, ca.id) AS to_currency_id, COALESCE(cb.decimals, ca.decimals) AS to_decimals
FROM transactions t
JOIN accounts a ON a.id = t.account_id
JOIN currencies ca ON ca.id = a.currency_id
LEFT JOIN accounts ta ON ta.id = t.to_account_id
LEFT JOIN currencies cb ON cb.id = ta.currency_id
WHERE t.id = ?;

-- name: ListTransactionSummaryRows :many
SELECT t.type, t.amount, c.rate, c.is_base, c.decimals
FROM transactions t
JOIN accounts a ON a.id = t.account_id
JOIN currencies c ON c.id = a.currency_id
WHERE t.status = ? AND t.type IN ('income','expense');

-- name: ListItemsOfTransactions :many
SELECT transaction_id, id, name, quantity, price_per_unit, total_price FROM transaction_items
WHERE transaction_id IN (sqlc.slice('ids'))
ORDER BY transaction_id, id;

-- name: ListTagsOfTransactions :many
SELECT tt.transaction_id, tags.id, tags.name, tags.created_at FROM tags
JOIN transaction_tag tt ON tt.tag_id = tags.id
WHERE tt.transaction_id IN (sqlc.slice('ids'))
ORDER BY tt.transaction_id, tags.name;

-- name: InsertTransactionItem :exec
INSERT INTO transaction_items (transaction_id, name, quantity, price_per_unit, total_price, created_at, updated_at)
VALUES (?,?,?,?,?,?,?);

-- name: DeleteTransactionTags :exec
DELETE FROM transaction_tag WHERE transaction_id = ?;

-- name: InsertTransactionTag :exec
INSERT OR IGNORE INTO transaction_tag (transaction_id, tag_id) VALUES (?,?);

-- name: InsertTransactionIgnoreDup :execresult
INSERT OR IGNORE INTO transactions (type, account_id, category_id, amount, description, date, status, dedup_hash, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?);

-- name: UpdateTransactionCategory :exec
UPDATE transactions SET category_id=?, updated_at=? WHERE id=?;

-- name: UpdateTransactionDescription :exec
UPDATE transactions SET description=?, updated_at=? WHERE id=?;
