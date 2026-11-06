// Package upgrade turns a space file of an older release into a current one.
// Each release whose schema cannot be migrated in place is one Hop, kept in a
// package of its own beside this one (v1, ...): it knows its frozen schema,
// how to recognise a file of that release, and how to write the next release's
// file from it. A file runs through the hops from its own release to the end.
//
// This package is the only door to them. The store asks Needed, which looks
// only at whether the current base is recorded, and hands any other file to
// Convert; the Laravel import asks for Entry. Supporting one more release is a
// new package and one entry in hops, and no caller changes.
package upgrade

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

	"savvy-go/internal/db"
	"savvy-go/internal/migrate"
	v1 "savvy-go/internal/migrate/upgrade/v1"
)

// ErrUnknownSchema is returned for a space file that is not at the current
// base and that no hop recognises.
var ErrUnknownSchema = errors.New("space file is of a schema this version cannot upgrade")

// Hop is one step of the chain.
type Hop struct {
	// Version is what a file of this release records in schema_migrations.
	Version string
	// Is reports whether the space file at path is of this release.
	Is func(ctx context.Context, path string) (bool, error)
	// Apply gives a database the frozen schema of this release, for reading
	// an untrusted file of it before it is converted.
	Apply func(ctx context.Context, db *sql.DB) error
	// EnsureIndexes creates the indexes of that schema which are missing.
	EnsureIndexes func(ctx context.Context, db *sql.DB) error
	// Convert writes the file of the next release at dst from the file at
	// src, which it only reads. dst must not exist.
	Convert func(ctx context.Context, src, dst string) error
}

// hops are in release order, oldest first.
var hops = []Hop{
	{Version: v1.Version, Is: v1.IsV1, Apply: v1.Apply, EnsureIndexes: v1.EnsureIndexes, Convert: v1.Convert},
}

// Entry is the release a Laravel import is shaped as: the oldest one. The
// import fills a file of that schema, and Convert takes it on from there, so
// a newer release never touches the import.
func Entry() Hop { return hops[0] }

// Needed reports whether the space file at path is not at the current base: it
// records space migrations, and the base is not among them. A file that was
// never migrated is not old, it is empty. Which release the file is of is not
// asked here; Convert finds out, or refuses.
func Needed(ctx context.Context, path string) (bool, error) {
	d, err := db.OpenReadOnly(path)
	if err != nil {
		return false, err
	}
	defer d.Close()
	return migrate.IsOldSpace(ctx, d)
}

// Versions are the versions of the old releases, which no migration of the
// current schema knows.
func Versions() []string {
	out := make([]string, len(hops))
	for i, h := range hops {
		out[i] = h.Version
	}
	return out
}

// Find returns the hop a space file starts at, or nil when the file is current
// or was never migrated.
func Find(ctx context.Context, path string) (*Hop, error) {
	_, h, err := find(ctx, path)
	return h, err
}

func find(ctx context.Context, path string) (int, *Hop, error) {
	for i := range hops {
		is, err := hops[i].Is(ctx, path)
		if err != nil {
			return 0, nil, err
		}
		if is {
			return i, &hops[i], nil
		}
	}
	return 0, nil, nil
}

// Convert writes the current file at dst from the old one at src, through
// every hop from the release of src on. src is only read. dst must not exist;
// when the conversion fails nothing is left at dst or beside it. A file that
// is not old is an error: callers ask Find first.
func Convert(ctx context.Context, src, dst string) error {
	first, hop, err := find(ctx, src)
	if err != nil {
		return err
	}
	if hop == nil {
		return fmt.Errorf("convert %s: %w", src, ErrUnknownSchema)
	}
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("convert space: %s already exists", dst)
	}
	from := src
	for i := first; i < len(hops); i++ {
		to := dst
		if i < len(hops)-1 {
			to = fmt.Sprintf("%s.hop%d", dst, i)
			defer db.RemoveFiles(to)
		}
		if err := hops[i].Convert(ctx, from, to); err != nil {
			return fmt.Errorf("convert from %s: %w", hops[i].Version, err)
		}
		from = to
	}
	return nil
}
