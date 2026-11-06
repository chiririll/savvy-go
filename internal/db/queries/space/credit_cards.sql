-- name: GetCreditCard :one
-- The same columns as ListCreditCards, so its row converts to ListCreditCardsRow.
SELECT a.id, a.name, a.type, a.is_service, a.currency_id, a.initial_balance, a.is_active, a.sort_order, a.created_at,
	k.credit_limit, k.cycle, k.statement_day, k.period_days, k.grace_days,
	c.id AS currency_id_join, c.code, c.name AS currency_name, c.symbol, c.decimals, c.is_base, c.rate
FROM credit_cards k
JOIN accounts a ON a.id = k.account_id
JOIN currencies c ON c.id = a.currency_id
WHERE k.account_id = ?;

-- name: ListCreditCards :many
-- only_active leaves out inactive cards.
SELECT a.id, a.name, a.type, a.is_service, a.currency_id, a.initial_balance, a.is_active, a.sort_order, a.created_at,
	k.credit_limit, k.cycle, k.statement_day, k.period_days, k.grace_days,
	c.id AS currency_id_join, c.code, c.name AS currency_name, c.symbol, c.decimals, c.is_base, c.rate
FROM credit_cards k
JOIN accounts a ON a.id = k.account_id
JOIN currencies c ON c.id = a.currency_id
WHERE a.is_active = COALESCE(sqlc.narg('only_active'), a.is_active)
ORDER BY a.sort_order, a.id;

-- name: UpsertCreditCard :exec
INSERT INTO credit_cards (account_id, credit_limit, cycle, statement_day, period_days, grace_days)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (account_id) DO UPDATE SET
	credit_limit = excluded.credit_limit, cycle = excluded.cycle, statement_day = excluded.statement_day,
	period_days = excluded.period_days, grace_days = excluded.grace_days;

-- name: DeleteCreditCardAccount :exec
-- The credit_cards row goes with the account (cascade).
DELETE FROM accounts WHERE id = ? AND type = 'credit';

-- name: AccountDailyFlows :many
-- Like AccountDailyDeltas, but also the money that came in (positive payments)
-- and went out (negative ones) per day: a credit card's statement reads what
-- was paid in and what was spent apart from the net change.
SELECT t.date AS day,
	CAST(SUM(p.amount) AS INTEGER) AS delta,
	CAST(SUM(CASE WHEN p.amount > 0 THEN p.amount ELSE 0 END) AS INTEGER) AS inflow,
	CAST(SUM(CASE WHEN p.amount < 0 THEN p.amount ELSE 0 END) AS INTEGER) AS outflow
FROM payments p
JOIN transactions t ON t.id = p.transaction_id
WHERE p.account_id = ? AND t.status = 'confirmed'
GROUP BY t.date
ORDER BY t.date;
