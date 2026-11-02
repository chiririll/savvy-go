-- name: InsertTransaction :execresult
INSERT INTO transactions (type, description, date, status, external_id, created_by, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?);

-- name: UpdateTransaction :exec
-- Status, external_id and created_by are never changed here. A move to another
-- day drops the manual place: it was a slot of the old day. The CASE reads the
-- old date, the SET list the new one.
UPDATE transactions SET type=?, description=?, date=?,
	day_order = CASE WHEN substr(date, 1, 10) IS substr(CAST(? AS TEXT), 1, 10) THEN day_order END,
	updated_at=? WHERE id=?;

-- name: DeleteTransaction :exec
-- Payments, lines and tags go with it (cascade).
DELETE FROM transactions WHERE id = ?;

-- name: ConfirmTransaction :exec
-- The real amounts are written to the payments by the caller.
UPDATE transactions SET status='confirmed', date=?,
	day_order = CASE WHEN substr(date, 1, 10) IS substr(CAST(? AS TEXT), 1, 10) THEN day_order END,
	updated_at=? WHERE id=?;

-- name: SkipTransaction :exec
UPDATE transactions SET status='skipped', updated_at=? WHERE id=?;

-- name: CountAllTransactions :one
SELECT COUNT(*) FROM transactions;

-- name: GetTransaction :one
-- Lists go through filter.ListTransactions, which selects the same columns in the
-- same order. The order differs from the table's, so sqlc makes a row type here
-- instead of reusing the model.
SELECT t.id, t.type, t.status, t.description, t.date, t.day_order, t.external_id, t.created_by,
	t.created_at, t.updated_at
FROM transactions t
WHERE t.id = ?;

-- name: ExistsTransactionExternalID :one
SELECT EXISTS (SELECT 1 FROM transactions WHERE external_id = ?);

-- name: UpdateTransactionDescription :exec
UPDATE transactions SET description=?, updated_at=? WHERE id=?;

-- name: TouchTransaction :exec
UPDATE transactions SET updated_at=? WHERE id=?;

-- name: ListTransactionDayOrder :many
-- The sort key of each id and its day, in the order the list shows them.
SELECT id, substr(date, 1, 10) AS day, COALESCE(day_order, id) AS sort_key
FROM transactions WHERE id IN (sqlc.slice('ids'));

-- name: SetTransactionDayOrder :exec
UPDATE transactions SET day_order = ? WHERE id = ?;

-- Payments

-- name: ListPaymentsOfTransactions :many
SELECT p.transaction_id, p.account_id, p.amount, p.space_transfer_uuid, a.type AS account_type, c.id AS currency_id, c.decimals
FROM payments p
JOIN accounts a ON a.id = p.account_id
JOIN currencies c ON c.id = a.currency_id
WHERE p.transaction_id IN (sqlc.slice('ids'))
ORDER BY p.transaction_id, p.account_id;

-- name: ListTransactionLinks :many
-- What ties a transaction to another module: the status of the space transfer
-- its payment on a space account belongs to ('' when none), and the recurring
-- rule that made it (0 when none).
SELECT t.id AS transaction_id,
       CAST(COALESCE((SELECT st.status FROM payments p JOIN space_transfers st ON st.uuid = p.space_transfer_uuid
                      WHERE p.transaction_id = t.id), '') AS TEXT) AS transfer_status,
       CAST(COALESCE((SELECT ri.recurring_transaction_id FROM recurring_instances ri
                      WHERE ri.transaction_id = t.id), 0) AS INTEGER) AS recurring_id
FROM transactions t
WHERE t.id IN (sqlc.slice('ids'));

-- name: ListPaymentAccountIDs :many
SELECT account_id FROM payments WHERE transaction_id = ?;

-- name: InsertPayment :exec
INSERT INTO payments (transaction_id, account_id, amount, space_transfer_uuid) VALUES (?,?,?,?);

-- name: UpdatePaymentAmount :exec
-- A payment is updated in place so that its space_transfer_uuid stays.
UPDATE payments SET amount = ? WHERE transaction_id = ? AND account_id = ?;

-- name: DeletePayment :exec
DELETE FROM payments WHERE transaction_id = ? AND account_id = ?;

-- Lines

-- name: ListLinesOfTransactions :many
SELECT l.transaction_id, l.seq, l.category_id, l.currency_id, l.amount, l.note, c.decimals
FROM transaction_lines l
JOIN currencies c ON c.id = l.currency_id
WHERE l.transaction_id IN (sqlc.slice('ids'))
ORDER BY l.transaction_id, l.seq;

-- name: DeleteTransactionLines :exec
DELETE FROM transaction_lines WHERE transaction_id = ?;

-- name: InsertTransactionLine :exec
INSERT INTO transaction_lines (transaction_id, seq, category_id, currency_id, amount, note) VALUES (?,?,?,?,?,?);

-- name: SetTransactionLinesCategory :exec
-- Gives a category to the lines that have none; a categorised line is the user's choice.
UPDATE transaction_lines SET category_id = ? WHERE transaction_id = ? AND category_id IS NULL;

-- Tags

-- name: ListTagsOfTransactions :many
SELECT tt.transaction_id, tags.id, tags.name, tags.created_at FROM tags
JOIN transaction_tag tt ON tt.tag_id = tags.id
WHERE tt.transaction_id IN (sqlc.slice('ids'))
ORDER BY tt.transaction_id, tags.name;

-- name: DeleteTransactionTags :exec
DELETE FROM transaction_tag WHERE transaction_id = ?;

-- name: InsertTransactionTag :exec
INSERT INTO transaction_tag (transaction_id, tag_id) VALUES (?,?) ON CONFLICT DO NOTHING;

