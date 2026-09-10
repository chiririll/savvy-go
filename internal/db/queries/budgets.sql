-- name: InsertBudget :execresult
INSERT INTO budgets (name, amount, currency_id, period, start_date, end_date,
	is_global, notify_at_percent, is_active, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?);

-- name: UpdateBudget :exec
UPDATE budgets SET name=?, amount=?, currency_id=?, period=?, start_date=?, end_date=?,
	is_global=?, notify_at_percent=?, is_active=?, updated_at=? WHERE id=?;

-- name: DeleteBudget :exec
DELETE FROM budgets WHERE id = ?;

-- name: ListBudgets :many
SELECT b.id, b.name, b.amount, b.currency_id, b.period, b.start_date, b.end_date,
	b.is_global, b.notify_at_percent, b.is_active
FROM budgets b
WHERE b.id = COALESCE(sqlc.narg('id'), b.id)
ORDER BY b.id;

-- name: ListBudgetCategories :many
SELECT c.id, c.name, c.type, c.icon, c.color, 0 AS transactions_count
FROM categories c JOIN budget_category bc ON bc.category_id = c.id
WHERE bc.budget_id = ?;

-- name: ListBudgetTags :many
SELECT tags.id, tags.name, tags.created_at, 0 AS transactions_count FROM tags
JOIN budget_tag bt ON bt.tag_id = tags.id WHERE bt.budget_id = ?;

-- name: DeleteBudgetCategories :exec
DELETE FROM budget_category WHERE budget_id = ?;

-- name: InsertBudgetCategory :exec
INSERT OR IGNORE INTO budget_category (budget_id, category_id) VALUES (?,?);

-- name: DeleteBudgetTags :exec
DELETE FROM budget_tag WHERE budget_id = ?;

-- name: InsertBudgetTag :exec
INSERT OR IGNORE INTO budget_tag (budget_id, tag_id) VALUES (?,?);

-- name: GetGlobalMonthlyBudget :one
SELECT b.amount, c.rate, c.is_base FROM budgets b
LEFT JOIN currencies c ON c.id = b.currency_id
WHERE b.is_active = 1 AND b.period = 'monthly' AND b.is_global = 1 LIMIT 1;

-- name: ListMonthlyBudgets :many
SELECT b.amount, c.rate, c.is_base FROM budgets b
LEFT JOIN currencies c ON c.id = b.currency_id
WHERE b.is_active = 1 AND b.period = 'monthly';
