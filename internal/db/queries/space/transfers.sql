-- name: ListSpaceTransfers :many
SELECT uuid, from_space_uuid, from_account_id, from_amount, from_currency, from_decimals,
	to_space_uuid, to_account_id, to_amount, to_currency, to_decimals,
	date, description, created_by, version, deleted_at, sig_v, signer_kid, signature,
	status, review, remote, updated_at
FROM space_transfers
ORDER BY date DESC, uuid DESC;

-- name: GetSpaceTransfer :one
SELECT uuid, from_space_uuid, from_account_id, from_amount, from_currency, from_decimals,
	to_space_uuid, to_account_id, to_amount, to_currency, to_decimals,
	date, description, created_by, version, deleted_at, sig_v, signer_kid, signature,
	status, review, remote, updated_at
FROM space_transfers WHERE uuid = ?;

-- name: UpsertSpaceTransfer :exec
INSERT INTO space_transfers (uuid, from_space_uuid, from_account_id, from_amount, from_currency, from_decimals,
	to_space_uuid, to_account_id, to_amount, to_currency, to_decimals,
	date, description, created_by, version, deleted_at, sig_v, signer_kid, signature,
	status, review, remote, updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT (uuid) DO UPDATE SET
	from_space_uuid = excluded.from_space_uuid, from_account_id = excluded.from_account_id,
	from_amount = excluded.from_amount, from_currency = excluded.from_currency, from_decimals = excluded.from_decimals,
	to_space_uuid = excluded.to_space_uuid, to_account_id = excluded.to_account_id,
	to_amount = excluded.to_amount, to_currency = excluded.to_currency, to_decimals = excluded.to_decimals,
	date = excluded.date, description = excluded.description, created_by = excluded.created_by,
	version = excluded.version, deleted_at = excluded.deleted_at, sig_v = excluded.sig_v,
	signer_kid = excluded.signer_kid, signature = excluded.signature,
	status = excluded.status, review = excluded.review, remote = excluded.remote, updated_at = excluded.updated_at;

-- name: DeleteSpaceTransfer :exec
DELETE FROM space_transfers WHERE uuid = ?;

-- name: SetSpaceTransferState :exec
UPDATE space_transfers SET status = ?, review = ?, remote = ?, updated_at = ? WHERE uuid = ?;

-- name: GetTransferTransaction :one
SELECT id, status FROM transactions WHERE space_transfer_uuid = ?;

-- name: InsertTransferTransaction :exec
INSERT INTO transactions (type, account_id, amount, description, date, status, space_transfer_uuid, created_by, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?);

-- name: UpdateTransferTransaction :exec
UPDATE transactions SET type = ?, account_id = ?, amount = ?, description = ?, date = ?, status = ?, updated_at = ?
WHERE space_transfer_uuid = ?;

-- name: DeleteTransferTransaction :exec
DELETE FROM transactions WHERE space_transfer_uuid = ?;

-- name: GetTransactionTransferUUID :one
SELECT space_transfer_uuid FROM transactions WHERE id = ?;

-- name: GetAccountCurrency :one
SELECT a.id, c.code, c.decimals FROM accounts a JOIN currencies c ON c.id = a.currency_id WHERE a.id = ?;
