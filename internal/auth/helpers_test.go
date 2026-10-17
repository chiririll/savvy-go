package auth

import (
	"context"
	"fmt"
	"testing"
	"time"

	"savvy-go/internal/db/dbtest"
	spacesrepo "savvy-go/internal/db/repo/spaces"
	"savvy-go/internal/store"
)

func ptr[T any](v T) *T { return &v }

func newDB(t *testing.T) store.DB {
	t.Helper()
	return dbtest.Server(t)
}

// newSpaces creates n space rows (tokens may only name spaces that exist) and returns their ids.
func newSpaces(t *testing.T, d store.DB, n int) []int64 {
	t.Helper()
	var ids []int64
	for i := range n {
		id, err := spacesrepo.Repo{DB: d}.Insert(context.Background(), fmt.Sprintf("uuid-%d", i), fmt.Sprintf("space %d", i), nil, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids
}

func newUser(t *testing.T, d store.DB, email, role string) *User {
	t.Helper()
	u, err := Users{DB: d}.Create(context.Background(), "Test", email, ptr("correct horse"), role)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
