package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	"savvy-go/internal/store"
)

// InSpaces locks the spaces in id order, opens a transaction on each, runs fn
// and commits them one after another. See store.Store.
func (s *Store) InSpaces(ctx context.Context, ids []int64, fn func(map[int64]store.DB) error) error {
	ids = slices.Clone(ids)
	slices.Sort(ids)
	ids = slices.Compact(ids)

	locked, err := s.lockSpaces(ids)
	defer unlockSpaces(locked)
	if err != nil {
		return err
	}
	txs, handles, err := beginAll(ctx, locked)
	if err != nil {
		return err
	}
	if err := fn(handles); err != nil {
		rollbackAll(txs)
		return mapErr(err)
	}
	return s.commitAll(txs)
}

// lockSpaces takes each space's write lock and read gate in order. It returns
// every space it locked, also on error, so the caller can unlock them.
func (s *Store) lockSpaces(ids []int64) ([]*space, error) {
	locked := make([]*space, 0, len(ids))
	for _, id := range ids {
		sp, err := s.lookup(id)
		if err != nil {
			return locked, fmt.Errorf("space %d: %w", id, err)
		}
		sp.write.Lock()
		sp.gate.RLock()
		locked = append(locked, sp)
		if sp.gone {
			return locked, fmt.Errorf("space %d: %w", id, store.ErrNotFound)
		}
	}
	return locked, nil
}

func unlockSpaces(locked []*space) {
	for i := len(locked) - 1; i >= 0; i-- {
		locked[i].gate.RUnlock()
		locked[i].write.Unlock()
	}
}

func rollbackAll(txs []*sql.Tx) {
	for _, tx := range txs {
		_ = tx.Rollback()
	}
}

// beginAll opens a transaction on each space, rolling back the ones already
// open if a later one fails.
func beginAll(ctx context.Context, locked []*space) ([]*sql.Tx, map[int64]store.DB, error) {
	txs := make([]*sql.Tx, 0, len(locked))
	handles := make(map[int64]store.DB, len(locked))
	for _, sp := range locked {
		tx, err := sp.db.BeginTx(ctx, nil)
		if err != nil {
			rollbackAll(txs)
			return nil, nil, err
		}
		txs = append(txs, tx)
		handles[sp.id] = txHandle{tx}
	}
	return txs, handles, nil
}

// commitAll commits the transactions in order; the first failure rolls back
// the rest.
func (s *Store) commitAll(txs []*sql.Tx) error {
	for i, tx := range txs {
		if err := tx.Commit(); err != nil {
			rollbackAll(txs[i+1:])
			return mapErr(err)
		}
		if s.opts.BetweenCommits != nil && i < len(txs)-1 {
			if err := s.opts.BetweenCommits(i + 1); err != nil {
				rollbackAll(txs[i+1:])
				return err
			}
		}
	}
	return nil
}
