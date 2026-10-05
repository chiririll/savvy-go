-- name: InsertUpload :exec
INSERT INTO uploads (id, user_id, bucket, object_key, disk, original_name, mime_type, size, part_size, total_parts, status, expires_at, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?);

-- name: GetUpload :one
SELECT id, user_id, bucket, object_key, disk, path, original_name, mime_type, size, part_size, total_parts, status, expires_at
FROM uploads WHERE id = ?;

-- name: CompleteUpload :exec
UPDATE uploads SET path=?, total_parts=?, status=?, completed_at=?, updated_at=? WHERE id=?;

-- name: SetUploadStatus :exec
UPDATE uploads SET status=?, updated_at=? WHERE id=?;

-- name: ListExpiredPendingUploads :many
SELECT id FROM uploads WHERE status = 'pending' AND expires_at IS NOT NULL AND expires_at < ?;

-- name: UploadUsageByBucket :many
SELECT bucket, status, COUNT(*), COALESCE(SUM(size), 0)
FROM uploads GROUP BY bucket, status;
