-- name: InsertRecurring :execresult
INSERT INTO recurring_transactions (template, schedule, is_active, error, created_at, updated_at)
VALUES (?,?,?,?,?,?);

-- name: UpdateRecurring :exec
-- Saving a rule replaces its error: the caller passes NULL once the rule is valid.
UPDATE recurring_transactions SET template=?, schedule=?, is_active=?, error=?, updated_at=? WHERE id=?;

-- name: StopRecurring :exec
UPDATE recurring_transactions SET is_active=0, error=?, updated_at=? WHERE id=?;

-- name: DeleteRecurring :exec
-- Its instances go with it (cascade); its transactions stay.
DELETE FROM recurring_transactions WHERE id = ?;

-- name: GetRecurring :one
-- The same columns as ListRecurring, so its row converts to ListRecurringRow.
-- last_due is the latest handled due date, whatever became of it; last_run the
-- latest one that was confirmed; pending_due the open occurrence's.
SELECT r.id, r.template, r.schedule, r.is_active, r.error, r.created_at, r.updated_at,
	(SELECT MAX(i.due_date) FROM recurring_instances i WHERE i.recurring_transaction_id = r.id) AS last_due,
	(SELECT MAX(i.due_date) FROM recurring_instances i
		JOIN transactions t ON t.id = i.transaction_id
		WHERE i.recurring_transaction_id = r.id AND t.status = 'confirmed') AS last_run,
	(SELECT MIN(i.due_date) FROM recurring_instances i
		JOIN transactions t ON t.id = i.transaction_id
		WHERE i.recurring_transaction_id = r.id AND t.status = 'pending') AS pending_due
FROM recurring_transactions r
WHERE r.id = ?;

-- name: ListRecurring :many
SELECT r.id, r.template, r.schedule, r.is_active, r.error, r.created_at, r.updated_at,
	(SELECT MAX(i.due_date) FROM recurring_instances i WHERE i.recurring_transaction_id = r.id) AS last_due,
	(SELECT MAX(i.due_date) FROM recurring_instances i
		JOIN transactions t ON t.id = i.transaction_id
		WHERE i.recurring_transaction_id = r.id AND t.status = 'confirmed') AS last_run,
	(SELECT MIN(i.due_date) FROM recurring_instances i
		JOIN transactions t ON t.id = i.transaction_id
		WHERE i.recurring_transaction_id = r.id AND t.status = 'pending') AS pending_due
FROM recurring_transactions r
WHERE r.is_active = COALESCE(sqlc.narg('active_only'), r.is_active)
ORDER BY r.id;

-- name: InsertRecurringInstance :execresult
-- A due date is handled once: a second insert for it changes nothing.
INSERT INTO recurring_instances (recurring_transaction_id, due_date, transaction_id) VALUES (?,?,?)
ON CONFLICT DO NOTHING;

-- name: GetPendingRecurringInstance :one
-- The open occurrence of a rule: an instance whose transaction is pending.
SELECT i.due_date, i.transaction_id FROM recurring_instances i
JOIN transactions t ON t.id = i.transaction_id
WHERE i.recurring_transaction_id = ? AND t.status = 'pending'
ORDER BY i.due_date LIMIT 1;

-- name: GetRecurringInstanceOf :one
-- The rule and due date a transaction was made for.
SELECT recurring_transaction_id, due_date FROM recurring_instances WHERE transaction_id = ?;

-- name: MoveRecurringInstance :execresult
-- The caller checks the new date is free: it is part of the key.
UPDATE recurring_instances SET due_date = ?
WHERE recurring_transaction_id = ? AND due_date = ?;

-- name: CountRecurringInstancesOn :one
SELECT COUNT(*) FROM recurring_instances WHERE recurring_transaction_id = ? AND due_date = ?;

-- name: ListRecurringInstances :many
SELECT due_date, transaction_id FROM recurring_instances
WHERE recurring_transaction_id = ? ORDER BY due_date;
