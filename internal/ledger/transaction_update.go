package ledger

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"savvy-go/internal/audit"
	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/store"
)

// Changing transactions, one or many at a time.

func (s Transactions) Update(ctx context.Context, id int64, in TxInput) (*Transaction, error) {
	return s.update(ctx, id, in, true)
}

// Modify is Update without reading the transaction back; see Add. It reports
// ErrUnknownTransaction for an id that does not exist.
func (s Transactions) Modify(ctx context.Context, id int64, in TxInput) error {
	tx, err := s.update(ctx, id, in, false)
	if err == nil && tx == nil {
		return ErrUnknownTransaction
	}
	return err
}

// update changes a transaction; read says whether to return it as stored.
// Without it the result is the transaction as it was, only to tell it exists.
func (s Transactions) update(ctx context.Context, id int64, in TxInput, read bool) (*Transaction, error) {
	p, cur, err := s.planUpdate(ctx, id, in)
	if err != nil || cur == nil {
		return cur, err
	}
	err = s.audited(ctx, id, audit.OpUpdate, func(d store.DB) error { return s.applyUpdate(ctx, d, p) })
	if err != nil {
		return nil, err
	}
	if !read {
		return cur, nil
	}
	tx, err := s.ByID(ctx, id)
	if tx != nil && p.changed {
		tx.Difference = s.difference(ctx, p.b)
	}
	return tx, err
}

// TxUpdate is a change to the transaction ID; what In leaves out is kept.
type TxUpdate struct {
	ID int64
	In TxInput
}

// UpdateMany changes all of ups in one database transaction and returns the
// transactions as stored, in the same order. Either all of them change or none:
// an update that does not validate, or names a transaction that does not exist
// or twice, fails the call before anything is written. The audit rows share a
// batch id, so one rollback undoes the call.
func (s Transactions) UpdateMany(ctx context.Context, ups []TxUpdate) ([]*Transaction, error) {
	seen := make(map[int64]bool, len(ups))
	plans := make([]updatePlan, len(ups))
	for i, u := range ups {
		if seen[u.ID] {
			return nil, fmt.Errorf("transaction %d: transaction %d is listed twice", i+1, u.ID)
		}
		seen[u.ID] = true
		p, cur, err := s.planUpdate(ctx, u.ID, u.In)
		if err != nil {
			return nil, fmt.Errorf("transaction %d: %w", i+1, err)
		}
		if cur == nil {
			return nil, fmt.Errorf("transaction %d: %w", i+1, ErrUnknownTransaction)
		}
		plans[i] = p
	}
	batch := uuid.NewString()
	err := store.Tx(ctx, s.DB, func(d store.DB) error {
		for i, p := range plans {
			before, err := (repo.Repo{DB: d}).Snapshot(ctx, p.id)
			if err != nil {
				return err
			}
			if err := s.applyUpdate(ctx, d, p); err != nil {
				return fmt.Errorf("transaction %d: %w", i+1, err)
			}
			if err := s.record(ctx, d, p.id, audit.OpUpdate, batch, before); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]*Transaction, len(plans))
	for i, p := range plans {
		tx, err := s.ByID(ctx, p.id)
		if err != nil {
			return nil, err
		}
		if tx != nil && p.changed {
			tx.Difference = s.difference(ctx, p.b)
		}
		out[i] = tx
	}
	return out, nil
}

// updatePlan is an update that passed validation and awaits its write.
type updatePlan struct {
	id      int64
	in      TxInput
	b       built
	changed bool
}

// planUpdate merges in into transaction id and checks the result. It reads
// only; the returned transaction is nil when id does not exist.
func (s Transactions) planUpdate(ctx context.Context, id int64, in TxInput) (updatePlan, *Transaction, error) {
	if in.ExternalID != nil || in.ExternalSource != nil {
		return updatePlan{}, nil, fmt.Errorf("external_id is set when a transaction is created and cannot be changed")
	}
	cur, err := s.byID(ctx, id, true)
	if err != nil || cur == nil {
		return updatePlan{}, cur, err
	}
	seen := cur.without(s.as)
	// What the caller leaves out is kept; an empty description clears it.
	merged := cur.Input()
	if in.Payments != nil {
		merged.Payments = in.Payments
	}
	if in.Lines != nil {
		merged.Lines = in.Lines
	}
	if in.Type != "" {
		merged.Type = in.Type
	}
	if in.Date != nil {
		merged.Date = in.Date
	}
	if in.Description != nil {
		merged.Description = in.Description
	}
	merged.TagIDs = in.TagIDs
	in = merged
	if seen.Status == "confirmed" && in.Date == nil {
		return updatePlan{}, nil, fmt.Errorf("date required")
	}
	b, err := s.As(s.ownerOf(*cur)).build(ctx, in, false)
	if err != nil {
		return updatePlan{}, nil, err
	}
	changed := !samePayments(cur.Payments, b.payments) || !sameLines(cur.Lines, b.lines, true)
	switch {
	case seen.Allowed().Edit:
		if err := frozenParts(seen, in, b); err != nil {
			return updatePlan{}, nil, err
		}
	case seen.Allowed().EditDetails:
		if in.Type != cur.Type || !sameDate(cur.Date, in.Date) || !samePayments(cur.Payments, b.payments) ||
			!sameLines(cur.Lines, b.lines, true) {
			return updatePlan{}, nil, ErrDetailsOnly
		}
	default:
		return updatePlan{}, nil, seen.refuse(ActionEdit)
	}
	return updatePlan{id: id, in: in, b: b, changed: changed}, cur, nil
}

// applyUpdate writes a planned update on d.
func (s Transactions) applyUpdate(ctx context.Context, d store.DB, p updatePlan) error {
	if err := (repo.Repo{DB: d}).UpdateTransaction(ctx, p.id, repo.TxFields{
		Type: p.in.Type, Description: blankToNil(p.in.Description), Date: p.in.Date,
	}, time.Now().UTC()); err != nil {
		return err
	}
	return s.store(ctx, d, p.id, p.b, p.in.TagIDs)
}
