package v1

import (
	"context"
	"database/sql"
	"encoding/json"
)

// payloadV1 is a frozen copy of the record internal/transfers signs with
// sig_v 1 (migrate sits below transfers and cannot import it). The stored
// bytes must equal what json.Marshal produced when the server signed: the
// same keys in the same order, nulls written out, and <, > and & escaped.
// Do not change it.
type payloadV1 struct {
	UUID        string  `json:"uuid"`
	From        sideV1  `json:"from"`
	To          sideV1  `json:"to"`
	Date        string  `json:"date"`
	Description *string `json:"description"`
	CreatedBy   *int64  `json:"created_by"`
	Version     int64   `json:"version"`
	DeletedAt   *string `json:"deleted_at"`
}

type sideV1 struct {
	SpaceUUID string `json:"space"`
	AccountID int64  `json:"account"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Decimals  int64  `json:"decimals"`
}

// recordV1 is the whole record as the other space's version is stored in
// remote: the payload with its signature.
type recordV1 struct {
	payloadV1
	SigV      int64  `json:"sig_v"`
	SignerKID string `json:"signer_kid"`
	Signature string `json:"signature"`
}

// oldTransfer is a v1.0 space_transfers row.
type oldTransfer struct {
	payload   payloadV1
	sigV      int64
	kid, sig  *string
	status    string
	review    *string
	remote    *string
	updatedAt *string
}

const readTransfers = `SELECT uuid, from_space_uuid, from_account_id, from_amount, from_currency, from_decimals,
	to_space_uuid, to_account_id, to_amount, to_currency, to_decimals, date, description, created_by, version,
	deleted_at, sig_v, signer_kid, signature, status, review, remote, updated_at
	FROM src.space_transfers ORDER BY uuid`

func loadTransfers(ctx context.Context, tx *sql.Tx) ([]oldTransfer, error) {
	rows, err := tx.QueryContext(ctx, readTransfers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []oldTransfer
	for rows.Next() {
		var t oldTransfer
		p := &t.payload
		if err := rows.Scan(&p.UUID,
			&p.From.SpaceUUID, &p.From.AccountID, &p.From.Amount, &p.From.Currency, &p.From.Decimals,
			&p.To.SpaceUUID, &p.To.AccountID, &p.To.Amount, &p.To.Currency, &p.To.Decimals,
			&p.Date, &p.Description, &p.CreatedBy, &p.Version, &p.DeletedAt,
			&t.sigV, &t.kid, &t.sig, &t.status, &t.review, &t.remote, &t.updatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// copyTransfers rebuilds each record's payload from its columns. ok and
// needs_attention become confirmed; pending becomes in_review, and a record
// that only the other space has (created_remote) moves into remote, with no
// payload of this space's own.
func copyTransfers(ctx context.Context, tx *sql.Tx) error {
	old, err := loadTransfers(ctx, tx)
	if err != nil {
		return err
	}
	for _, t := range old {
		if err := insertTransfer(ctx, tx, t); err != nil {
			return err
		}
	}
	return nil
}

func insertTransfer(ctx context.Context, tx *sql.Tx, t oldTransfer) error {
	status := "confirmed"
	var payload, remote *string
	kid, sig := nonEmpty(t.kid), nonEmpty(t.sig)
	switch {
	case t.status != "pending":
		payload = marshal(t.payload)
	case t.review != nil && *t.review == "created_remote":
		status, kid, sig = "in_review", nil, nil
		remote = marshal(recordV1{t.payload, t.sigV, deref(t.kid), deref(t.sig)})
	default:
		status, payload, remote = "in_review", marshal(t.payload), t.remote
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO main.space_transfers
		(uuid, payload, sig_v, source_kid, source_sig, status, remote, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		t.payload.UUID, payload, t.sigV, kid, sig, status, remote, t.updatedAt)
	return err
}

func marshal(v any) *string {
	b, err := json.Marshal(v)
	if err != nil {
		return nil // the types here are plain structs; this cannot happen
	}
	s := string(b)
	return &s
}

func nonEmpty(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
