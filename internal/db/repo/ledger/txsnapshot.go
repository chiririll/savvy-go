package ledger

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/money"
)

// TxSnapshot is a transaction as stored, exact and self-contained: what the
// audit log keeps to be able to put it back. Money is minor units, with the
// currency of each line; a payment has no currency of its own, it is the
// currency of its account.
type TxSnapshot struct {
	ID             int64             `json:"id"`
	Type           string            `json:"type"`
	Status         string            `json:"status"`
	Description    *string           `json:"description,omitempty"`
	Date           *string           `json:"date,omitempty"`
	DayOrder       *int64            `json:"day_order,omitempty"`
	ExternalSource *string           `json:"external_source,omitempty"`
	ExternalID     *string           `json:"external_id,omitempty"`
	CreatedBy      *int64            `json:"created_by,omitempty"`
	CreatedAt      *time.Time        `json:"created_at,omitempty"`
	UpdatedAt      *time.Time        `json:"updated_at,omitempty"`
	Payments       []PaymentSnapshot `json:"payments"`
	Lines          []LineSnapshot    `json:"lines"`
	TagIDs         []int64           `json:"tag_ids"`
}

// PaymentSnapshot is a payment: signed minor units of the account's currency.
type PaymentSnapshot struct {
	AccountID         int64   `json:"account_id"`
	Minor             int64   `json:"minor"`
	SpaceTransferUUID *string `json:"space_transfer_uuid,omitempty"`
}

// LineSnapshot is a line: positive minor units of the currency CurrencyID.
type LineSnapshot struct {
	CategoryID *int64  `json:"category_id,omitempty"`
	CurrencyID int64   `json:"currency_id"`
	Minor      int64   `json:"minor"`
	Note       *string `json:"note,omitempty"`
}

// Snapshot reads a transaction as stored; nil for an unknown id.
func (r Repo) Snapshot(ctx context.Context, id int64) (*TxSnapshot, error) {
	tx, err := r.Transaction(ctx, id)
	if err != nil || tx == nil {
		return nil, err
	}
	return snapshotOf(*tx), nil
}

// Snapshots reads many transactions as stored, in the order of ids, in a few
// queries instead of a few per transaction. An unknown id is a nil entry.
func (r Repo) Snapshots(ctx context.Context, ids []int64) ([]*TxSnapshot, error) {
	out := make([]*TxSnapshot, len(ids))
	for from := 0; from < len(ids); from += snapshotChunk {
		to := min(from+snapshotChunk, len(ids))
		rows, err := r.q().GetTransactionsByIDs(ctx, ids[from:to])
		if err != nil {
			return nil, err
		}
		txs := make([]Tx, len(rows))
		at := make(map[int64]int, len(rows))
		for i, x := range rows {
			txs[i] = txOf(sqlc.GetTransactionRow(x))
			at[x.ID] = i
		}
		if err := r.load(ctx, txs); err != nil {
			return nil, err
		}
		for i := from; i < to; i++ {
			if j, ok := at[ids[i]]; ok {
				out[i] = snapshotOf(txs[j])
			}
		}
	}
	return out, nil
}

// snapshotChunk is how many ids one query takes, well under SQLite's limit on
// bound variables.
const snapshotChunk = 500

func snapshotOf(tx Tx) *TxSnapshot {
	s := &TxSnapshot{
		ID: tx.ID, Type: tx.Type, Status: tx.Status, Description: tx.Description, Date: tx.Date, DayOrder: tx.DayOrder,
		ExternalSource: tx.ExternalSource, ExternalID: tx.ExternalID, CreatedBy: tx.CreatedBy,
		CreatedAt: tx.CreatedAt, UpdatedAt: tx.UpdatedAt,
		Payments: []PaymentSnapshot{}, Lines: []LineSnapshot{}, TagIDs: []int64{},
	}
	for _, p := range tx.Payments {
		s.Payments = append(s.Payments, PaymentSnapshot{
			AccountID: p.AccountID, Minor: p.Amount.Minor(), SpaceTransferUUID: p.SpaceTransferUUID,
		})
	}
	for _, l := range tx.Lines {
		s.Lines = append(s.Lines, LineSnapshot{
			CategoryID: l.CategoryID, CurrencyID: l.Amount.Unit().ID, Minor: l.Amount.Minor(), Note: l.Note,
		})
	}
	for _, t := range tx.Tags {
		s.TagIDs = append(s.TagIDs, t.ID)
	}
	return s
}

// Restore puts a transaction back exactly as the snapshot has it: the row is
// updated when it exists and inserted under its own id when it does not, then
// its payments, lines and tags are made the snapshot's. It checks nothing but
// what the database does (foreign keys, the unique external key): the caller
// has made sure nothing changed since the snapshot was taken.
func (r Repo) Restore(ctx context.Context, s TxSnapshot, now time.Time) error {
	have, err := r.Transaction(ctx, s.ID)
	if err != nil {
		return err
	}
	if have == nil {
		if err := r.q().InsertTransactionWithID(ctx, sqlc.InsertTransactionWithIDParams{
			ID: s.ID, Type: s.Type, Description: db.NullString(s.Description), Date: db.NullString(s.Date),
			Status: s.Status, DayOrder: db.NullInt64(s.DayOrder),
			ExternalSource: db.NullString(s.ExternalSource), ExternalID: db.NullString(s.ExternalID),
			CreatedBy: db.NullInt64(s.CreatedBy), CreatedAt: db.NullTime(s.CreatedAt), UpdatedAt: db.TimeNS(now),
		}); err != nil {
			return err
		}
	} else if err := r.q().RestoreTransactionRow(ctx, sqlc.RestoreTransactionRowParams{
		Type: s.Type, Description: db.NullString(s.Description), Date: db.NullString(s.Date), Status: s.Status,
		DayOrder: db.NullInt64(s.DayOrder), UpdatedAt: db.TimeNS(now), ID: s.ID,
	}); err != nil {
		return err
	}
	payments := make([]PaymentFields, len(s.Payments))
	for i, p := range s.Payments {
		payments[i] = PaymentFields{AccountID: p.AccountID, Amount: money.New(p.Minor, money.Unit{}), SpaceTransferUUID: p.SpaceTransferUUID}
	}
	if err := r.ReplacePayments(ctx, s.ID, payments); err != nil {
		return err
	}
	lines := make([]LineFields, len(s.Lines))
	for i, l := range s.Lines {
		lines[i] = LineFields{CategoryID: l.CategoryID, Amount: money.New(l.Minor, money.Unit{ID: l.CurrencyID}), Note: l.Note}
	}
	if err := r.ReplaceLines(ctx, s.ID, lines); err != nil {
		return err
	}
	return r.SetTags(ctx, s.ID, s.TagIDs)
}
