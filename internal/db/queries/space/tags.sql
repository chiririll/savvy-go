-- name: ListTags :many
SELECT tags.id, tags.name, tags.created_at,
	(SELECT COUNT(*) FROM transaction_tag tt WHERE tt.tag_id = tags.id)
FROM tags ORDER BY name;

-- name: GetTag :one
SELECT tags.id, tags.name, tags.created_at,
	(SELECT COUNT(*) FROM transaction_tag tt WHERE tt.tag_id = tags.id)
FROM tags WHERE tags.id = ?;

-- name: InsertTag :execresult
INSERT INTO tags (name, created_at, updated_at) VALUES (?,?,?);

-- name: UpdateTag :exec
UPDATE tags SET name=?, updated_at=? WHERE id=?;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = ?;
