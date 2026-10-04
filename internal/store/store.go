// Package store is the storage boundary. Business logic talks to the server
// database and to one database per space through these interfaces and never
// learns how they are laid out (one SQLite file per space today, possibly one
// Postgres schema per space later).
package store

import (
	"context"
	"database/sql"
	"errors"
)

// DB is a database handle: a whole database or a transaction on it. It is
// the method set sqlc's DBTX needs, so *sql.DB and *sql.Tx satisfy it.
type DB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Txer is implemented by handles that run work in a transaction themselves.
type Txer interface {
	Tx(ctx context.Context, fn func(DB) error) error
}

var (
	// ErrQuotaExceeded is returned when a write would grow a space beyond its quota.
	ErrQuotaExceeded = errors.New("space storage quota exceeded")
	// ErrUnavailable is returned for a space whose database could not be
	// opened or migrated; other spaces keep working.
	ErrUnavailable = errors.New("space unavailable")
	// ErrNotFound is returned for a space the store does not hold.
	ErrNotFound = errors.New("space not found")
)

// Tx runs fn in a transaction on db. A handle that is already a transaction
// runs fn on itself, so code that needs atomicity composes with callers that
// already opened one (store.InSpaces, an outer Tx).
func Tx(ctx context.Context, db DB, fn func(DB) error) error {
	switch d := db.(type) {
	case Txer:
		return d.Tx(ctx, fn)
	case *sql.Tx:
		return fn(d)
	case *sql.DB:
		tx, err := d.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback() }()
		if err := fn(tx); err != nil {
			return err
		}
		return tx.Commit()
	default:
		return errors.New("store: handle cannot start a transaction")
	}
}

// Store holds the server database and the space databases.
type Store interface {
	// Server is the database of users, memberships and everything else that
	// is not inside a space.
	Server() DB
	// Space returns the database of one space.
	Space(ctx context.Context, id int64) (DB, error)
	// InSpaces runs fn with one transactional handle per space and commits
	// them together. Locks are taken in id order, so concurrent calls never
	// deadlock. A failure inside fn rolls every space back; a crash between
	// the commits can leave them apart, so every write made through InSpaces
	// must be versioned and idempotent for a later merge to reconcile.
	InSpaces(ctx context.Context, ids []int64, fn func(map[int64]DB) error) error
	// CreateSpace creates and migrates the database of a new space.
	CreateSpace(ctx context.Context, id int64) error
	// DeleteSpace removes the database of a space.
	DeleteSpace(ctx context.Context, id int64) error
	// Spaces lists the spaces the store holds, ascending.
	Spaces(ctx context.Context) ([]int64, error)
	// SpaceSize is the storage a space occupies, in bytes.
	SpaceSize(ctx context.Context, id int64) (int64, error)
	// ApplyQuota makes a changed size limit of a space take effect.
	ApplyQuota(ctx context.Context, id int64) error
	// Status reports startup progress and spaces that are unavailable.
	Status() Status
	Close() error

	// Backups move whole databases in and out. Files from outside are never
	// used as they are: Prepare* rebuilds them from the store's own schema
	// and validates them, and only a prepared artifact can replace or create
	// a database. Discard removes a prepared artifact that was not used.

	// ExportSpace writes a consistent copy of a space database to dest.
	ExportSpace(ctx context.Context, id int64, dest string) error
	// ExportServer writes the server database and every space database into
	// dir; spaces cannot be created or removed meanwhile.
	ExportServer(ctx context.Context, dir string) error
	// PrepareSpace validates a space database (a space backup, a single-file
	// or Laravel-era database) for ReplaceSpace or ImportSpace.
	PrepareSpace(ctx context.Context, src string) (*PreparedSpace, error)
	// PrepareServer validates a server backup: a directory written by
	// ExportServer, or a single-file or Laravel-era database, which is split.
	PrepareServer(ctx context.Context, src string) (*PreparedServer, error)
	// ReplaceSpace swaps a space's database for a prepared one.
	ReplaceSpace(ctx context.Context, id int64, p *PreparedSpace) error
	// ImportSpace creates space id from a prepared database.
	ImportSpace(ctx context.Context, id int64, p *PreparedSpace) error
	// ReplaceServer swaps the server database and every space for a prepared
	// server backup.
	ReplaceServer(ctx context.Context, p *PreparedServer) error
	// Discard removes a prepared artifact that will not be used.
	Discard(artifact string)
}

// PreparedSpace is a validated space database. Artifact belongs to the store;
// callers only hand it back.
type PreparedSpace struct {
	Artifact string
	UUID     string // the space_uuid recorded in the database, "" if none
	Size     int64  // bytes
	// Settings are written into the space's settings before the database
	// goes live, so a mark set here survives a crash during the swap.
	Settings map[string]string
}

// PreparedServer is a validated server backup.
type PreparedServer struct {
	Artifact string
	Spaces   []int64
}

// Status is what the readiness probe and the admin overview show.
type Status struct {
	Ready       bool
	Unavailable map[int64]string // space id -> reason
}
