-- name: ListCurrencies :many
SELECT id, code, name, symbol, decimals, is_base, rate FROM currencies ORDER BY code;

-- name: GetCurrency :one
SELECT id, code, name, symbol, decimals, is_base, rate FROM currencies WHERE id = ?;

-- name: GetCurrencyByCode :one
SELECT id, code, name, symbol, decimals, is_base, rate FROM currencies WHERE code = ?;

-- name: GetBaseCurrency :one
SELECT id, code, name, symbol, decimals, is_base, rate FROM currencies WHERE is_base = 1 LIMIT 1;

-- name: GetBaseCurrencyCode :one
SELECT code FROM currencies WHERE is_base = 1 LIMIT 1;

-- name: ClearBaseCurrency :exec
UPDATE currencies SET is_base = 0 WHERE is_base = 1;

-- name: InsertCurrency :execresult
INSERT INTO currencies (code, name, symbol, decimals, is_base, rate, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateCurrency :exec
UPDATE currencies SET name=?, symbol=?, decimals=?, is_base=?, rate=?, updated_at=? WHERE id=?;

-- name: CountAccountsForCurrency :one
SELECT COUNT(*) FROM accounts WHERE currency_id = ?;

-- name: CountCurrencies :one
SELECT COUNT(*) FROM currencies;

-- name: DeleteCurrency :exec
DELETE FROM currencies WHERE id = ?;

-- name: ListOtherCurrencyRates :many
SELECT id, rate FROM currencies WHERE id != ?;

-- name: UpdateCurrencyRate :exec
UPDATE currencies SET rate = ?, is_base = 0 WHERE id = ?;

-- name: SetCurrencyBase :exec
UPDATE currencies SET is_base = 1, rate = 1, updated_at = ? WHERE id = ?;

-- name: ListCurrencyCodes :many
SELECT code FROM currencies;
