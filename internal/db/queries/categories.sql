-- name: ListCategories :many
SELECT c.id, c.name, c.type, c.icon, c.color, c.is_default,
	(SELECT COUNT(*) FROM transactions t WHERE t.category_id = c.id)
FROM categories c
WHERE c.type = COALESCE(sqlc.narg('type'), c.type)
ORDER BY c.name;

-- name: GetCategory :one
SELECT categories.id, categories.name, categories.type, categories.icon, categories.color, categories.is_default,
	(SELECT COUNT(*) FROM transactions t WHERE t.category_id = categories.id)
FROM categories WHERE categories.id = ?;

-- name: InsertCategory :execresult
INSERT INTO categories (name, type, icon, color, is_default, created_at, updated_at) VALUES (?,?,?,?,?,?,?);

-- name: UpdateCategory :exec
UPDATE categories SET name=?, type=?, icon=?, color=?, is_default=?, updated_at=? WHERE id=?;

-- name: CountCategoriesByType :one
SELECT COUNT(*) FROM categories WHERE type = ?;

-- name: DeleteCategory :exec
DELETE FROM categories WHERE id = ?;

-- name: CategoryStatistics :one
SELECT COUNT(*), COALESCE(SUM(amount),0) FROM transactions
WHERE category_id = sqlc.arg('category_id') AND status = 'confirmed'
  AND date >= COALESCE(sqlc.narg('start_date'), date)
  AND date <= COALESCE(sqlc.narg('end_date'), date);

-- name: GetDefaultCategory :one
SELECT categories.id, categories.name, categories.type, categories.icon, categories.color, categories.is_default,
	(SELECT COUNT(*) FROM transactions t WHERE t.category_id = categories.id)
FROM categories WHERE categories.type = ? AND categories.is_default = 1 LIMIT 1;

-- name: ClearDefaultCategory :exec
UPDATE categories SET is_default = 0 WHERE type = ? AND is_default = 1;

-- name: SetCategoryDefault :exec
UPDATE categories SET is_default = 1, updated_at = ? WHERE id = ?;

-- name: ReassignCategoryTransactions :exec
UPDATE transactions SET category_id = ? WHERE category_id = ?;
