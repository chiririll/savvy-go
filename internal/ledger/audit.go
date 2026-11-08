package ledger

import (
	"context"
	"reflect"

	"savvy-go/internal/audit"
	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/store"
)

// EntityTransaction is the entity type of a transaction in the audit log.
const EntityTransaction = "transaction"

// record writes the audit row of a change to transaction id on d, the handle
// of the database transaction that made it. before is how the transaction
// stood (nil: it did not exist); how it stands now is read here. A change that
// left it as it was (an update that sent the same values) writes nothing.
func (s Transactions) record(
	ctx context.Context, d store.DB, id int64, op, batch string, before *repo.TxSnapshot,
) error {
	after, err := (repo.Repo{DB: d}).Snapshot(ctx, id)
	if err != nil {
		return err
	}
	if before == nil && after == nil || before != nil && after != nil && sameSnapshot(*before, *after) {
		return nil
	}
	_, err = audit.Log{}.Record(ctx, d, audit.Change{
		EntityType: EntityTransaction, EntityID: id, Op: op, BatchID: batch,
		Before: audit.Snapshot(before), After: audit.Snapshot(after),
	})
	return err
}

// recordCreated writes the audit rows of transactions ids, which were just
// created on d in one call, reading them back in a few queries for all of
// them instead of a few for each.
func (s Transactions) recordCreated(ctx context.Context, d store.DB, ids []int64, batch string) error {
	afters, err := (repo.Repo{DB: d}).Snapshots(ctx, ids)
	if err != nil {
		return err
	}
	for i, after := range afters {
		if after == nil {
			continue
		}
		if _, err := (audit.Log{}).Record(ctx, d, audit.Change{
			EntityType: EntityTransaction, EntityID: ids[i], Op: audit.OpCreate, BatchID: batch, After: audit.Snapshot(after),
		}); err != nil {
			return err
		}
	}
	return nil
}

// sameSnapshot compares two snapshots by content: the time of the last write
// is not a change.
func sameSnapshot(a, b repo.TxSnapshot) bool {
	a.UpdatedAt, b.UpdatedAt = nil, nil
	return reflect.DeepEqual(a, b)
}

// audited runs change on a transaction in a database transaction of its own
// and logs it as op with the transaction's snapshots before and after.
func (s Transactions) audited(ctx context.Context, id int64, op string, change func(d store.DB) error) error {
	return store.Tx(ctx, s.DB, func(d store.DB) error {
		before, err := (repo.Repo{DB: d}).Snapshot(ctx, id)
		if err != nil {
			return err
		}
		if err := change(d); err != nil {
			return err
		}
		return s.record(ctx, d, id, op, "", before)
	})
}
