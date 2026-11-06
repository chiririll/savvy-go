-- name: ListCategories :many
SELECT c.id, c.name, c.type, c.icon, c.color, c.is_default,
	(SELECT COUNT(DISTINCT l.transaction_id) FROM transaction_lines l WHERE l.category_id = c.id)
FROM categories c
WHERE c.type = COALESCE(sqlc.narg('type'), c.type)
ORDER BY c.name;

-- name: GetCategory :one
SELECT categories.id, categories.name, categories.type, categories.icon, categories.color, categories.is_default,
	(SELECT COUNT(DISTINCT l.transaction_id) FROM transaction_lines l WHERE l.category_id = categories.id)
FROM categories WHERE categories.id = ?;

-- name: InsertCategory :execresult
INSERT INTO categories (name, type, icon, color, is_default, created_at, updated_at) VALUES (?,?,?,?,?,?,?);

-- name: UpdateCategory :exec
UPDATE categories SET name=?, type=?, icon=?, color=?, is_default=?, updated_at=? WHERE id=?;

-- name: CountCategoriesByType :one
SELECT COUNT(*) FROM categories WHERE type = ?;

-- name: DeleteCategory :exec
DELETE FROM categories WHERE id = ?;

-- name: CategoryStatistics :many
-- Per currency of the lines: how many transactions, and what their lines in
-- this category add up to. A transaction counts once however many of its lines
-- are in the category.
SELECT COUNT(DISTINCT t.id) AS cnt, CAST(COALESCE(SUM(l.amount),0) AS INTEGER) AS total, c.decimals, c.rate
FROM transaction_lines l
JOIN transactions t ON t.id = l.transaction_id
JOIN currencies c ON c.id = l.currency_id
WHERE l.category_id = sqlc.arg('category_id') AND t.status = 'confirmed'
  AND t.date >= COALESCE(sqlc.narg('start_date'), t.date)
  AND t.date <= COALESCE(sqlc.narg('end_date'), t.date)
GROUP BY c.id;

-- name: GetDefaultCategory :one
SELECT categories.id, categories.name, categories.type, categories.icon, categories.color, categories.is_default,
	(SELECT COUNT(DISTINCT l.transaction_id) FROM transaction_lines l WHERE l.category_id = categories.id)
FROM categories WHERE categories.type = ? AND categories.is_default = 1 LIMIT 1;

-- name: ClearDefaultCategory :exec
UPDATE categories SET is_default = 0 WHERE type = ? AND is_default = 1;

-- name: SetCategoryDefault :exec
UPDATE categories SET is_default = 1, updated_at = ? WHERE id = ?;

-- name: ReassignCategoryTransactions :exec
-- Every line of the old category moves to the new one.
UPDATE transaction_lines SET category_id = ? WHERE category_id = ?;
