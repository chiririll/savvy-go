-- name: InsertImport :exec
INSERT INTO transaction_imports (id, user_id, upload_id, status, created_at, updated_at)
VALUES (?,?,?,?,?,?);

-- name: GetImport :one
SELECT id, user_id, upload_id, status, mapping, options, total_rows, processed_rows,
	created_count, skipped_count, error_count, errors, meta, message
FROM transaction_imports WHERE id = ?;

-- name: MarkImportParsed :exec
UPDATE transaction_imports SET status='parsed', total_rows=?, meta=?, updated_at=? WHERE id=?;

-- name: MarkImportImporting :exec
UPDATE transaction_imports SET status='importing', mapping=?, options=?, processed_rows=0,
	created_count=0, skipped_count=0, error_count=0, errors=NULL, message=NULL, updated_at=? WHERE id=?;

-- name: MarkImportCompleted :exec
UPDATE transaction_imports SET status='completed', processed_rows=?, created_count=?, skipped_count=?,
	error_count=?, errors=?, meta=?, updated_at=? WHERE id=?;

-- name: FailImport :exec
UPDATE transaction_imports SET status='failed', message=?, updated_at=? WHERE id=?;

-- name: ImportCountsByStatus :many
SELECT status, COUNT(*) FROM transaction_imports GROUP BY status;
