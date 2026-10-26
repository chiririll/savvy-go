package ledger

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/store"
)

// A group is one payment made in several parts: with several accounts (cash
// and card) or in several currencies (euros, with the change in dinars). Its
// legs are ordinary transactions that share a group_uuid, so balances, budgets
// and reports see them as they always have. A payment with change is an
// expense of the price plus a transfer that exchanges the surplus, so the
// expense never depends on today's rate.

// ErrGroupLeg is returned for a change to one leg of a payment: the group
// changes, not its legs.
var ErrGroupLeg = errors.New("this transaction is one leg of a payment; change the payment instead")

// ErrGroupInvalid is returned for legs that are not one payment.
var ErrGroupInvalid = errors.New("invalid payment")

// ErrGroupNotFound is returned for a group that does not exist.
var ErrGroupNotFound = errors.New("payment not found")

// Group is the legs of one payment, oldest first.
type Group struct {
	UUID string
	Legs []Transaction
}

type Groups struct{ DB store.DB }

func (s Groups) txs(db store.DB) Transactions { return Transactions{DB: db} }

// Create writes every leg or none.
func (s Groups) Create(ctx context.Context, legs []TxInput) (*Group, error) {
	if err := validateLegs(legs); err != nil {
		return nil, err
	}
	v7, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	id := v7.String()
	err = store.Tx(ctx, s.DB, func(d store.DB) error { return s.insert(ctx, d, id, legs) })
	if err != nil {
		return nil, err
	}
	return s.ByUUID(ctx, id)
}

// Update replaces the legs of a payment, all in one step. The legs get new
// ids; the payment keeps its uuid.
func (s Groups) Update(ctx context.Context, id string, legs []TxInput) (*Group, error) {
	if err := validateLegs(legs); err != nil {
		return nil, err
	}
	err := store.Tx(ctx, s.DB, func(d store.DB) error {
		r := repo.Repo{DB: d}
		ids, err := r.GroupTransactionIDs(ctx, id)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			return ErrGroupNotFound
		}
		if err := r.DeleteGroup(ctx, id); err != nil {
			return err
		}
		return s.insert(ctx, d, id, legs)
	})
	if err != nil {
		return nil, err
	}
	return s.ByUUID(ctx, id)
}

// Delete removes every leg.
func (s Groups) Delete(ctx context.Context, id string) error {
	return store.Tx(ctx, s.DB, func(d store.DB) error {
		r := repo.Repo{DB: d}
		ids, err := r.GroupTransactionIDs(ctx, id)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			return ErrGroupNotFound
		}
		return r.DeleteGroup(ctx, id)
	})
}

// ByUUID returns nil for an unknown payment.
func (s Groups) ByUUID(ctx context.Context, id string) (*Group, error) {
	ids, err := (repo.Repo{DB: s.DB}).GroupTransactionIDs(ctx, id)
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	g := &Group{UUID: id}
	for _, txID := range ids {
		leg, err := s.txs(s.DB).ByID(ctx, txID)
		if err != nil {
			return nil, err
		}
		if leg != nil {
			g.Legs = append(g.Legs, *leg)
		}
	}
	return g, nil
}

func (s Groups) insert(ctx context.Context, d store.DB, id string, legs []TxInput) error {
	for _, leg := range legs {
		leg.group = &id
		if _, err := s.txs(d).Create(ctx, leg); err != nil {
			return err
		}
	}
	return nil
}

// validateLegs checks what makes legs one payment: at least two, all settled
// on the same past or present day, and only the types that move an
// account's own money.
func validateLegs(legs []TxInput) error {
	if len(legs) < 2 {
		return fmt.Errorf("%w: a payment needs at least two legs", ErrGroupInvalid)
	}
	for _, leg := range legs {
		switch leg.Type {
		case "expense", "income", "transfer":
		default:
			return fmt.Errorf("%w: a %q cannot be a leg", ErrGroupInvalid, leg.Type)
		}
		if leg.Status != nil && *leg.Status != "confirmed" {
			return fmt.Errorf("%w: a leg must be confirmed", ErrGroupInvalid)
		}
		if leg.IsEstimated {
			return fmt.Errorf("%w: a leg cannot be an estimate", ErrGroupInvalid)
		}
		if leg.Date == nil || *leg.Date != *legs[0].Date {
			return fmt.Errorf("%w: every leg needs the same date", ErrGroupInvalid)
		}
		if IsFuture(*leg.Date) {
			return fmt.Errorf("%w: a leg cannot be in the future", ErrGroupInvalid)
		}
	}
	return nil
}
