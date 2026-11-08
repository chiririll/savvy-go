package transfers

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
)

// Mirror is this space's side of a transfer as it appears in the books: a
// transaction with a payment on a real account and the opposite payment on the
// space account of that account's currency, which carries the uuid.
type Mirror struct {
	UUID, Type, Date, Status string
	// AccountID is the real account and Amount what it moved, signed.
	AccountID, Amount int64
	Description       *string
	CreatedBy         *int64
}

// HasMirror reports whether this side's transaction exists.
func (r Space) HasMirror(ctx context.Context, uuid string) (bool, error) {
	_, err := r.q().GetTransferMirror(ctx, db.NS(uuid))
	if err == db.ErrNotFound {
		return false, nil
	}
	return err == nil, err
}

// DeleteMirror removes the transaction of this side; having none is no error.
func (r Space) DeleteMirror(ctx context.Context, uuid string) error {
	return r.q().DeleteTransferMirror(ctx, db.NS(uuid))
}

// SaveMirror makes the transaction of this side the given one, creating the
// space account of the currency when it does not exist. Payments are updated
// in place; one whose account no longer takes part (the real account was
// changed to one of another currency) is deleted before its replacement is
// inserted, so the uuid is free again. The real account must exist.
func (r Space) SaveMirror(ctx context.Context, m Mirror, now time.Time) error {
	acc, err := r.q().GetAccountCurrency(ctx, m.AccountID)
	if err != nil {
		return err
	}
	space, err := r.spaceAccount(ctx, acc.CurrencyID, acc.Code, now)
	if err != nil {
		return err
	}
	id, err := r.mirrorTransaction(ctx, m, now)
	if err != nil {
		return err
	}
	want := map[int64]int64{m.AccountID: m.Amount, space: -m.Amount}
	have, err := r.q().ListPaymentAccountIDs(ctx, id)
	if err != nil {
		return err
	}
	had := map[int64]bool{}
	for _, a := range have {
		had[a] = true
		if _, ok := want[a]; !ok {
			if err := r.q().DeletePayment(ctx, sqlc.DeletePaymentParams{TransactionID: id, AccountID: a}); err != nil {
				return err
			}
		}
	}
	for a, amount := range want {
		if had[a] {
			err = r.q().UpdatePaymentAmount(ctx,
				sqlc.UpdatePaymentAmountParams{Amount: amount, TransactionID: id, AccountID: a})
		} else {
			var uuid db.NString
			if a == space {
				uuid = db.NS(m.UUID)
			}
			err = r.q().InsertPayment(ctx,
				sqlc.InsertPaymentParams{TransactionID: id, AccountID: a, Amount: amount, SpaceTransferUuid: uuid})
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// mirrorTransaction writes the transaction row of a mirror and returns its id.
func (r Space) mirrorTransaction(ctx context.Context, m Mirror, now time.Time) (int64, error) {
	cur, err := r.q().GetTransferMirror(ctx, db.NS(m.UUID))
	if err == db.ErrNotFound {
		res, err := r.q().InsertTransaction(ctx, sqlc.InsertTransactionParams{
			Type: m.Type, Description: db.NullString(m.Description), Date: db.NS(m.Date), Status: m.Status,
			CreatedBy: db.NullInt64(m.CreatedBy), CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
		})
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	if err != nil {
		return 0, err
	}
	if err := r.q().UpdateTransaction(ctx, sqlc.UpdateTransactionParams{
		Type: m.Type, Description: db.NullString(m.Description), Date: db.NS(m.Date),
		Column4: m.Date, UpdatedAt: db.TimeNS(now), ID: cur.ID,
	}); err != nil {
		return 0, err
	}
	if cur.Status != m.Status {
		if err := r.q().SetTransferMirrorStatus(ctx,
			sqlc.SetTransferMirrorStatusParams{Status: m.Status, UpdatedAt: db.TimeNS(now), ID: cur.ID}); err != nil {
			return 0, err
		}
	}
	return cur.ID, nil
}

// spaceAccount is the account of type space for a currency, made when missing.
func (r Space) spaceAccount(ctx context.Context, currencyID int64, code string, now time.Time) (int64, error) {
	id, err := r.q().GetSpaceAccountID(ctx, currencyID)
	if err == nil {
		return id, nil
	}
	if err != db.ErrNotFound {
		return 0, err
	}
	res, err := r.q().InsertAccount(ctx, sqlc.InsertAccountParams{
		Name: "Other spaces " + code, Type: "space", CurrencyID: currencyID, IsActive: 1,
		CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// AccountCurrency is the currency code and decimals of an account, and whether
// it is a service account (a debt or a space account), which a transfer cannot
// take as its own side.
type AccountCurrency struct {
	Code      string
	Decimals  int64
	IsService bool
}

// AccountCurrency returns nil when the account does not exist.
func (r Space) AccountCurrency(ctx context.Context, account int64) (*AccountCurrency, error) {
	row, err := r.q().GetAccountCurrency(ctx, account)
	if err == db.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &AccountCurrency{
		Code: row.Code, Decimals: row.Decimals, IsService: row.Type == "debt" || row.Type == "space",
	}, nil
}

// ReviewKey marks a restored space whose transfers wait for a review merge;
// it is written before the restored file goes live.
const ReviewKey = "transfer_review_pending"

// ReviewPending reports whether the space still waits for its review merge.
func (r Space) ReviewPending(ctx context.Context) (bool, error) {
	v, err := r.q().GetSpaceSetting(ctx, ReviewKey)
	if err != nil {
		if err == db.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return v.Valid && v.String != "", nil
}

// ClearReview removes the review mark; a missing mark is not an error.
func (r Space) ClearReview(ctx context.Context) error {
	return r.q().DeleteSpaceSetting(ctx, ReviewKey)
}
