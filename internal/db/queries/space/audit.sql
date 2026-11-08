-- name: InsertAuditLog :execresult
INSERT INTO audit_log (at, source, user_id, token_id, token_label, entity_type, entity_id, op,
	before_json, after_json, batch_id, details)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?);

-- name: GetAuditLog :one
SELECT * FROM audit_log WHERE id = ?;

-- name: ListAuditLog :many
-- Newest first. Every filter is optional; before_id pages (the caller passes the largest id for the first page).
SELECT * FROM audit_log
WHERE id < sqlc.arg(before_id)
	AND token_id IS COALESCE(sqlc.narg(token_id), token_id)
	AND entity_type = COALESCE(sqlc.narg(entity_type), entity_type)
	AND COALESCE(entity_id, 0) = COALESCE(sqlc.narg(entity_id), COALESCE(entity_id, 0))
	AND op = COALESCE(sqlc.narg(op), op)
	AND source = COALESCE(sqlc.narg(source), source)
ORDER BY id DESC
LIMIT sqlc.arg(row_limit);

-- name: ListAuditLogUser :many
-- Like ListAuditLog, for the rows of one user whatever the door.
SELECT * FROM audit_log
WHERE id < sqlc.arg(before_id)
	AND user_id = sqlc.arg(user_id)
	AND entity_type = COALESCE(sqlc.narg(entity_type), entity_type)
	AND op = COALESCE(sqlc.narg(op), op)
	AND source = COALESCE(sqlc.narg(source), source)
ORDER BY id DESC
LIMIT sqlc.arg(row_limit);

-- name: ListAuditLogBatch :many
SELECT * FROM audit_log WHERE batch_id = ? ORDER BY id DESC;

-- name: FirstAuditOverlap :one
-- The first row after row `after_id` that changed the same entity and stands:
-- not undone itself, and not a rollback row (the undone row carries that).
SELECT id FROM audit_log
WHERE entity_type = ? AND entity_id = ? AND id > ? AND rolled_back_by IS NULL AND op <> 'rollback'
ORDER BY id LIMIT 1;

-- name: MarkAuditRolledBack :exec
UPDATE audit_log SET rolled_back_by = ? WHERE id = ? AND rolled_back_by IS NULL;

-- name: DeleteAuditLogUpTo :execresult
DELETE FROM audit_log WHERE id <= ?;

-- name: DeleteAuditLogOfTokenUpTo :execresult
DELETE FROM audit_log WHERE id <= ? AND token_id = ?;

-- name: MaxAuditLogID :one
SELECT CAST(COALESCE(MAX(id), 0) AS INTEGER) FROM audit_log;

-- name: ListAuditStanding :many
-- The standing rows after row `after_id` that changed any of the entities, as
-- FirstAuditOverlap counts them, oldest first: one query for a page of rows.
SELECT id, entity_id FROM audit_log
WHERE entity_type = ? AND entity_id IN (sqlc.slice('entity_ids')) AND id > ?
  AND rolled_back_by IS NULL AND op <> 'rollback'
ORDER BY id;
