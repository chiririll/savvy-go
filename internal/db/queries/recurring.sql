-- name: InsertRecurring :execresult
INSERT INTO recurring_transactions (
	type, account_id, to_account_id, category_id, amount, to_amount, description,
	frequency, interval, day_of_week, day_of_month, start_date, end_date,
	next_run_date, is_active, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?);

-- name: UpdateRecurring :exec
UPDATE recurring_transactions SET type=?, account_id=?, to_account_id=?, category_id=?,
	amount=?, to_amount=?, description=?, frequency=?, interval=?, day_of_week=?,
	day_of_month=?, start_date=?, end_date=?, next_run_date=?, is_active=?, updated_at=?
WHERE id=?;

-- name: DeletePendingForRecurring :exec
DELETE FROM transactions WHERE recurring_transaction_id = ? AND status = 'pending';

-- name: DeleteRecurring :exec
DELETE FROM recurring_transactions WHERE id = ?;

-- name: AdvanceRecurring :exec
UPDATE recurring_transactions SET last_run_date=?, next_run_date=?, updated_at=? WHERE id=?;

-- name: GetPendingRecurringTx :one
SELECT id FROM transactions WHERE recurring_transaction_id = ? AND status = 'pending' LIMIT 1;

-- name: DeleteRecurringTags :exec
DELETE FROM recurring_transaction_tag WHERE recurring_transaction_id = ?;

-- name: InsertRecurringTag :exec
INSERT OR IGNORE INTO recurring_transaction_tag (recurring_transaction_id, tag_id) VALUES (?,?);

-- name: ListRecurring :many
SELECT id, type, account_id, to_account_id, category_id, amount, to_amount, description,
	frequency, interval, day_of_week, day_of_month, start_date, end_date,
	next_run_date, last_run_date, is_active
FROM recurring_transactions
WHERE id = COALESCE(sqlc.narg('id'), id)
  AND is_active = COALESCE(sqlc.narg('active_only'), is_active)
ORDER BY next_run_date, id
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: ListRecurringTags :many
SELECT tags.id, tags.name, tags.created_at, 0 AS transactions_count FROM tags
JOIN recurring_transaction_tag rt ON rt.tag_id = tags.id
WHERE rt.recurring_transaction_id = ?;
