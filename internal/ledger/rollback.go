package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"savvy-go/internal/audit"
	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/store"
)

// Rollbacks puts back what the audit log recorded. The rules every entity
// shares (whose entries, nothing changed since) are internal/audit's; the
// inverse of each entity's changes is here, because only the ledger knows how
// to write it. A rollback is a restore of the snapshot, not a replay through
// the services, so it needs no grant the original change did not: a token that
// may add transactions but not delete them still takes its own create back.
// The snapshot was valid when it was taken and nothing has changed the entity
// since, which the log guarantees.
type Rollbacks struct{ DB store.DB }

// Log is the audit log of the space.
func (r Rollbacks) Log() audit.Log { return audit.Log{DB: r.DB} }

// supported says whether the ledger knows an inverse for e.
func supported(e audit.Entry) error {
	if e.EntityType != EntityTransaction {
		return fmt.Errorf("changes to %ss", e.EntityType)
	}
	switch e.Op {
	case audit.OpCreate, audit.OpUpdate, audit.OpDelete, audit.OpConfirm, audit.OpSkip:
		return nil
	}
	return fmt.Errorf("a %s entry", e.Op)
}

// Undoable says whether e could be rolled back now by the holder of tokenID
// (0: anyone entitled to), and if not, why.
func (r Rollbacks) Undoable(ctx context.Context, e audit.Entry, tokenID int64) error {
	return r.Log().Undoable(ctx, e, tokenID, supported)
}

// UndoableAll is Undoable for a page of entries: errs[i] answers es[i].
func (r Rollbacks) UndoableAll(ctx context.Context, es []audit.Entry, tokenID int64) ([]error, error) {
	return r.Log().UndoableAll(ctx, es, tokenID, supported)
}

// Undo rolls back the entries ids, newest first, all or none. tokenID limits
// it to that token's own entries (0: no limit, for a space admin). It returns
// the ids of the rollback rows.
func (r Rollbacks) Undo(ctx context.Context, ids []int64, tokenID int64) ([]int64, error) {
	rolled, err := r.Log().Undo(ctx, ids, tokenID, supported, inverse)
	if errors.Is(err, store.ErrConstraint) {
		// Foreign keys are checked when the database transaction commits, so
		// this is where a restore that points at a deleted row fails.
		err = fmt.Errorf("an account, category or tag it used is gone: %w", err)
	}
	return rolled, err
}

// UndoBatch rolls back every entry of a bulk call that still stands.
func (r Rollbacks) UndoBatch(ctx context.Context, batchID string, tokenID int64) ([]int64, error) {
	entries, err := r.Log().Batch(ctx, batchID)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, audit.ErrNotFound
	}
	ids := make([]int64, len(entries))
	for i, e := range entries {
		ids[i] = e.ID
	}
	return r.Undo(ctx, ids, tokenID)
}

// inverse undoes e, a change to a transaction, on d.
func inverse(ctx context.Context, d store.DB, e audit.Entry) (json.RawMessage, json.RawMessage, error) {
	r := repo.Repo{DB: d}
	id := *e.EntityID
	cur, err := r.Snapshot(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if e.Op == audit.OpCreate {
		if cur == nil {
			return nil, nil, fmt.Errorf("%w: the transaction is gone", audit.ErrNotUndoable)
		}
		if err := r.DeleteTransaction(ctx, id); err != nil {
			return nil, nil, err
		}
		return audit.Snapshot(cur), nil, nil
	}
	var want repo.TxSnapshot
	if err := json.Unmarshal(e.Before, &want); err != nil {
		return nil, nil, fmt.Errorf("%w: its snapshot is unreadable", audit.ErrNotUndoable)
	}
	if cur == nil && want.ExternalID != nil && want.ExternalSource != nil {
		// The key is unique by rule, not by constraint: look before putting it back.
		if other, found, err := r.IDByExternalID(ctx, *want.ExternalSource, *want.ExternalID); err != nil {
			return nil, nil, err
		} else if found {
			return nil, nil, &ExternalIDExistsError{Source: *want.ExternalSource, ExternalID: *want.ExternalID, TransactionID: other}
		}
	}
	if err := r.Restore(ctx, want, time.Now().UTC()); err != nil {
		return nil, nil, err
	}
	now, err := r.Snapshot(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return audit.Snapshot(cur), audit.Snapshot(now), nil
}
