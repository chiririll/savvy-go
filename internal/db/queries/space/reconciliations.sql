-- name: UpsertReconciliation :one
-- One point per account and date: a second write for the same date replaces
-- the balance and note and keeps the id and created_at.
INSERT INTO reconciliations (account_id, date, actual_balance, note, created_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (account_id, date) DO UPDATE SET actual_balance = excluded.actual_balance, note = excluded.note
RETURNING id, account_id, date, actual_balance, note, created_at;

-- name: GetReconciliation :one
SELECT id, account_id, date, actual_balance, note, created_at FROM reconciliations WHERE id = ?;

-- name: DeleteReconciliation :execrows
DELETE FROM reconciliations WHERE id = ?;

-- name: ListAccountReconciliations :many
-- Oldest first.
SELECT id, account_id, date, actual_balance, note, created_at FROM reconciliations
WHERE account_id = ? ORDER BY date, id;

-- name: ListReconciliationsBetween :many
-- Points of all accounts in a date range (inclusive, both optional), oldest
-- first; the caller keeps the accounts it wants.
SELECT id, account_id, date, actual_balance, note, created_at FROM reconciliations
WHERE date >= COALESCE(sqlc.narg('date_from'), date) AND date <= COALESCE(sqlc.narg('date_to'), date)
ORDER BY date, account_id, id;

-- name: LatestReconciliationAsOf :one
-- The newest point of an account dated on or before as_of.
SELECT id, account_id, date, actual_balance, note, created_at FROM reconciliations
WHERE account_id = ? AND date <= ? ORDER BY date DESC LIMIT 1;

-- name: RestoreReconciliation :one
-- Puts a point back under its own id (a rollback of a delete), or overwrites
-- the one that has the id (a rollback of an update).
INSERT INTO reconciliations (id, account_id, date, actual_balance, note, created_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET date = excluded.date, actual_balance = excluded.actual_balance, note = excluded.note
RETURNING id, account_id, date, actual_balance, note, created_at;
