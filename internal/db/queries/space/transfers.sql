-- name: ListSpaceTransfers :many
-- source, destination and version are generated from payload (or remote).
SELECT uuid, payload, sig_v, source_kid, source_sig, destination_kid, destination_sig,
	source, destination, version, status, remote, updated_at
FROM space_transfers
ORDER BY COALESCE(payload, remote) ->> '$.date' DESC, uuid DESC;

-- name: GetSpaceTransfer :one
SELECT uuid, payload, sig_v, source_kid, source_sig, destination_kid, destination_sig,
	source, destination, version, status, remote, updated_at
FROM space_transfers WHERE uuid = ?;

-- name: UpsertSpaceTransfer :exec
INSERT INTO space_transfers (uuid, payload, sig_v, source_kid, source_sig, destination_kid, destination_sig,
	status, remote, updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?)
ON CONFLICT (uuid) DO UPDATE SET
	payload = excluded.payload, sig_v = excluded.sig_v,
	source_kid = excluded.source_kid, source_sig = excluded.source_sig,
	destination_kid = excluded.destination_kid, destination_sig = excluded.destination_sig,
	status = excluded.status, remote = excluded.remote, updated_at = excluded.updated_at;

-- name: DeleteSpaceTransfer :exec
-- Its mirror transaction goes first: a payment restricts the delete.
DELETE FROM space_transfers WHERE uuid = ?;

-- Mirror: the transaction of this space's side. Its payment on the space
-- account carries the uuid.

-- name: GetTransferMirror :one
SELECT t.id, t.status FROM transactions t
JOIN payments p ON p.transaction_id = t.id
WHERE p.space_transfer_uuid = ?;

-- name: SetTransferMirrorStatus :exec
UPDATE transactions SET status = ?, updated_at = ? WHERE id = ?;

-- name: DeleteTransferMirror :exec
-- Its payments go with it (cascade).
DELETE FROM transactions WHERE id IN (SELECT transaction_id FROM payments WHERE space_transfer_uuid = ?);

-- name: GetAccountCurrency :one
SELECT a.id, a.type, c.id AS currency_id, c.code, c.decimals
FROM accounts a JOIN currencies c ON c.id = a.currency_id WHERE a.id = ?;

-- Keys of this space: a retired key still verifies.

-- name: ListSpaceKeys :many
SELECT kid, seed, created_at, retired_at FROM space_keys ORDER BY created_at, kid;

-- name: InsertSpaceKey :exec
INSERT INTO space_keys (kid, seed, created_at) VALUES (?, ?, ?);

-- name: RetireSpaceKeys :exec
UPDATE space_keys SET retired_at = ? WHERE retired_at IS NULL;
