package ledger

import (
	"context"
	"time"

	"savvy-go/internal/audit"
	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/store"
)

// Narrow changes made by automation actions.

// SetCategory gives a category to the uncategorised lines of a transaction (an
// automation action). The category must fit the transaction type, as for a
// line entered by hand.
func (s Transactions) SetCategory(ctx context.Context, id, categoryID int64) error {
	tx, err := s.ByID(ctx, id)
	if err != nil || tx == nil {
		return err
	}
	cat, err := (Categories{DB: s.DB}).ByID(ctx, categoryID)
	if err != nil {
		return err
	}
	if cat == nil {
		return ErrUnknownCategory
	}
	if cat.Type != lineCategoryType(tx.Type) {
		return ErrLineCategory
	}
	return s.audited(ctx, id, audit.OpUpdate, func(d store.DB) error {
		return (repo.Repo{DB: d}).SetTransactionCategory(ctx, id, categoryID, time.Now().UTC())
	})
}

// SetDescription changes only a transaction's description (an automation action).
func (s Transactions) SetDescription(ctx context.Context, id int64, description string) error {
	return s.audited(ctx, id, audit.OpUpdate, func(d store.DB) error {
		return (repo.Repo{DB: d}).SetTransactionDescription(ctx, id, description, time.Now().UTC())
	})
}

// SetTags replaces the tags of a transaction.
func (s Transactions) SetTags(ctx context.Context, txID int64, ids []int64) error {
	return s.audited(ctx, txID, audit.OpUpdate, func(d store.DB) error {
		return (repo.Repo{DB: d}).SetTags(ctx, txID, ids)
	})
}
