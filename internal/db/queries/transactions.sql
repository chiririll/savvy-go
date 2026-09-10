-- name: InsertTransaction :execresult
INSERT INTO transactions (type, account_id, to_account_id, category_id, amount, to_amount, exchange_rate,
	description, date, status, recurring_transaction_id, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?);

-- name: UpdateTransaction :exec
UPDATE transactions SET type=?, account_id=?, to_account_id=?, category_id=?, amount=?, to_amount=?,
	exchange_rate=?, description=?, date=?, updated_at=? WHERE id=?;

-- name: DeleteTransactionItems :exec
DELETE FROM transaction_items WHERE transaction_id = ?;

-- name: DeleteTransaction :exec
DELETE FROM transactions WHERE id = ?;

-- name: ConfirmTransaction :exec
UPDATE transactions SET status='confirmed', date=?, updated_at=? WHERE id=?;

-- name: SkipTransaction :exec
UPDATE transactions SET status='skipped', updated_at=? WHERE id=?;

-- Optional filters use each narg once (COALESCE/CASE). sqlc sqlite emits ?NNN;
-- modernc.org/sqlite counts every '?' so "OR col = ?N" (second use) over-binds.

-- name: CountTransactions :one
SELECT COUNT(*) FROM transactions t
WHERE t.id = COALESCE(sqlc.narg('id'), t.id)
  AND t.type = COALESCE(sqlc.narg('type'), t.type)
  AND t.account_id = COALESCE(sqlc.narg('account_id'), t.account_id)
  AND IFNULL(t.category_id, -1) = IFNULL(sqlc.narg('category_id'), IFNULL(t.category_id, -1))
  AND t.status = COALESCE(sqlc.narg('status'), t.status)
  AND t.date >= COALESCE(sqlc.narg('start_date'), t.date)
  AND t.date <= COALESCE(sqlc.narg('end_date'), t.date);

-- name: ListTransactions :many
SELECT t.id, t.type, t.account_id, t.to_account_id, t.category_id, t.amount, t.to_amount, t.exchange_rate,
	t.description, t.date, t.status, t.recurring_transaction_id, t.created_at
FROM transactions t
WHERE t.id = COALESCE(sqlc.narg('id'), t.id)
  AND t.type = COALESCE(sqlc.narg('type'), t.type)
  AND t.account_id = COALESCE(sqlc.narg('account_id'), t.account_id)
  AND IFNULL(t.category_id, -1) = IFNULL(sqlc.narg('category_id'), IFNULL(t.category_id, -1))
  AND t.status = COALESCE(sqlc.narg('status'), t.status)
  AND t.date >= COALESCE(sqlc.narg('start_date'), t.date)
  AND t.date <= COALESCE(sqlc.narg('end_date'), t.date)
ORDER BY t.date DESC, t.id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: ListTransactionSummaryRows :many
SELECT t.type, t.amount, c.rate, c.is_base
FROM transactions t
JOIN accounts a ON a.id = t.account_id
JOIN currencies c ON c.id = a.currency_id
WHERE t.status = ? AND t.type IN ('income','expense');

-- name: ListTransactionItems :many
SELECT id, name, quantity, price_per_unit, total_price FROM transaction_items WHERE transaction_id = ?;

-- name: ListTransactionTags :many
SELECT tags.id, tags.name, tags.created_at, 0 AS transactions_count FROM tags
JOIN transaction_tag tt ON tt.tag_id = tags.id WHERE tt.transaction_id = ?;

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
