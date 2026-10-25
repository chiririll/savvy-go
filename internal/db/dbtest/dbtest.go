// Package dbtest opens real, migrated databases for the tests of the
// repositories in internal/db: a fresh server database or space database in
// a temporary directory, removed when the test ends.
package dbtest

import (
	"context"
	"testing"

	"savvy-go/internal/store"
	"savvy-go/internal/store/sqlite"
	"savvy-go/internal/store/sqlite/sqlitetest"
)

func open(t *testing.T) *sqlite.Store { return sqlitetest.Open(t) }

// Server is a fresh server database.
func Server(t *testing.T) store.DB { return open(t).Server() }

// Space is a fresh space database.
func Space(t *testing.T) store.DB {
	t.Helper()
	st := open(t)
	ctx := context.Background()
	if err := st.CreateSpace(ctx, 1); err != nil {
		t.Fatal(err)
	}
	d, err := st.Space(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
