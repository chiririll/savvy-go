package sqlite

import (
	"context"
	"database/sql"

	"savvy-go/internal/store"
)

// handle is the store.DB of one space. Each statement holds the space gate
// shared for its duration, so a restore or delete waits for running
// statements and later statements see the new file. Closing the old
// *sql.DB waits for rows still being read.
type handle struct{ sp *space }

func (h handle) db() (*sql.DB, func(), error) {
	h.sp.gate.RLock()
	if h.sp.gone {
		h.sp.gate.RUnlock()
		return nil, nil, store.ErrNotFound
	}
	return h.sp.db, h.sp.gate.RUnlock, nil
}

func (h handle) ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error) {
	d, done, err := h.db()
	if err != nil {
		return nil, err
	}
	defer done()
	res, err := d.ExecContext(ctx, q, args...)
	return res, mapErr(err)
}

func (h handle) PrepareContext(ctx context.Context, q string) (*sql.Stmt, error) {
	d, done, err := h.db()
	if err != nil {
		return nil, err
	}
	defer done()
	return d.PrepareContext(ctx, q)
}

func (h handle) QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	d, done, err := h.db()
	if err != nil {
		return nil, err
	}
	defer done()
	return d.QueryContext(ctx, q, args...)
}

// QueryRowContext cannot report a gone space before Scan; it then queries a
// closed database, whose error Scan returns.
func (h handle) QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row {
	h.sp.gate.RLock()
	defer h.sp.gate.RUnlock()
	return h.sp.db.QueryRowContext(ctx, q, args...)
}

// Tx runs fn in a transaction on this space. fn must use the handle it is
// given: the space has one connection, which the transaction holds.
func (h handle) Tx(ctx context.Context, fn func(store.DB) error) error {
	d, done, err := h.db()
	if err != nil {
		return err
	}
	defer done()
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(txHandle{tx}); err != nil {
		return mapErr(err)
	}
	return mapErr(tx.Commit())
}

// txHandle is a transaction handed to InSpaces or Tx callbacks; a nested Tx
// joins it.
type txHandle struct{ tx *sql.Tx }

func (t txHandle) ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error) {
	res, err := t.tx.ExecContext(ctx, q, args...)
	return res, mapErr(err)
}

func (t txHandle) PrepareContext(ctx context.Context, q string) (*sql.Stmt, error) {
	return t.tx.PrepareContext(ctx, q)
}

func (t txHandle) QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, q, args...)
}

func (t txHandle) QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row {
	return t.tx.QueryRowContext(ctx, q, args...)
}

func (t txHandle) Tx(_ context.Context, fn func(store.DB) error) error {
	return fn(t)
}
