package ledger

import (
	"context"
	"fmt"
	"slices"
	"time"

	"savvy-go/internal/audit"
	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/store"
)

// Deleting, confirming, skipping and copying transactions.

func (s Transactions) Delete(ctx context.Context, id int64) error {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return err
	}
	if err := cur.without(s.as).refuse(ActionDelete); err != nil {
		return err
	}
	return s.audited(ctx, id, audit.OpDelete, func(d store.DB) error {
		return (repo.Repo{DB: d}).DeleteTransaction(ctx, id)
	})
}

// Amendment replaces the payments and lines of a pending occurrence when it
// is confirmed: its amounts were a guess (a rate of the day, a bill not yet
// known). Left nil, they stand.
type Amendment struct {
	Payments []PaymentInput
	Lines    []LineInput
}

// Confirm settles a pending transaction on date. Only the open occurrence of
// a recurring rule can be confirmed with other amounts.
func (s Transactions) Confirm(ctx context.Context, id int64, date string, amend *Amendment) (*Transaction, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	seen := cur.without(s.as)
	if err := seen.refuse(ActionConfirm); err != nil {
		return nil, err
	}
	if date == "" && cur.Date != nil {
		date = *cur.Date
	}
	if date == "" {
		return nil, fmt.Errorf("date required")
	}
	if IsFuture(date) {
		return nil, fmt.Errorf("future")
	}
	var b built
	amended := amend != nil && (amend.Payments != nil || amend.Lines != nil)
	if amended {
		if !slices.Contains(cur.Reasons, ReasonRecurringPending) {
			return nil, fmt.Errorf("only a scheduled occurrence can be confirmed with other amounts")
		}
		in := cur.Input()
		if amend.Payments != nil {
			in.Payments = amend.Payments
		}
		if amend.Lines != nil {
			in.Lines = amend.Lines
		}
		if b, err = s.As(s.ownerOf(*cur)).build(ctx, in, false); err != nil {
			return nil, err
		}
	}
	err = s.audited(ctx, id, audit.OpConfirm, func(d store.DB) error {
		r := repo.Repo{DB: d}
		if amended {
			if err := r.ReplacePayments(ctx, id, b.payments); err != nil {
				return err
			}
			if err := r.ReplaceLines(ctx, id, b.lines); err != nil {
				return err
			}
		}
		return r.ConfirmTransaction(ctx, id, date, time.Now().UTC())
	})
	if err != nil {
		return nil, err
	}
	tx, err := s.ByID(ctx, id)
	if tx != nil && amended {
		tx.Difference = s.difference(ctx, b)
	}
	return tx, err
}

func (s Transactions) Skip(ctx context.Context, id int64) (*Transaction, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if err := cur.without(s.as).refuse(ActionSkip); err != nil {
		return nil, err
	}
	err = s.audited(ctx, id, audit.OpSkip, func(d store.DB) error {
		return (repo.Repo{DB: d}).SkipTransaction(ctx, id, time.Now().UTC())
	})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

// Duplicate copies a transaction as a confirmed one dated today.
func (s Transactions) Duplicate(ctx context.Context, id int64) (*Transaction, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if err := cur.without(s.as).refuse(ActionDuplicate); err != nil {
		return nil, err
	}
	in := cur.Input()
	today := Today()
	st := "confirmed"
	in.Date, in.Status = &today, &st
	return Transactions{DB: s.DB}.Create(ctx, in)
}
