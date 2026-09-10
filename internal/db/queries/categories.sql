-- name: ListCategories :many
SELECT c.id, c.name, c.type, c.icon, c.color,
	(SELECT COUNT(*) FROM transactions t WHERE t.category_id = c.id)
FROM categories c
WHERE c.type = COALESCE(sqlc.narg('type'), c.type)
ORDER BY c.name;

-- name: GetCategory :one
SELECT categories.id, categories.name, categories.type, categories.icon, categories.color,
	(SELECT COUNT(*) FROM transactions t WHERE t.category_id = categories.id)
FROM categories WHERE categories.id = ?;

-- name: InsertCategory :execresult
INSERT INTO categories (name, type, icon, color, created_at, updated_at) VALUES (?,?,?,?,?,?);

-- name: UpdateCategory :exec
UPDATE categories SET name=?, type=?, icon=?, color=?, updated_at=? WHERE id=?;

-- name: CountCategoriesByType :one
SELECT COUNT(*) FROM categories WHERE type = ?;

-- name: DeleteCategory :exec
DELETE FROM categories WHERE id = ?;

-- name: CategoryStatistics :one
SELECT COUNT(*), COALESCE(SUM(amount),0) FROM transactions
WHERE category_id = sqlc.arg('category_id') AND status = 'confirmed'
  AND date >= COALESCE(sqlc.narg('start_date'), date)
  AND date <= COALESCE(sqlc.narg('end_date'), date);
