package spacekeys_test

import (
	"context"
	"testing"
	"time"

	"savvy-go/internal/db/dbtest"
	"savvy-go/internal/db/repo/spacekeys"
)

func TestAddRetiresTheCurrentKey(t *testing.T) {
	ctx := context.Background()
	r := spacekeys.Repo{DB: dbtest.Space(t)}
	if keys, err := r.List(ctx); err != nil || len(keys) != 0 {
		t.Fatalf("new space keys = %v, %v", keys, err)
	}
	t0 := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	if err := r.Add(ctx, "k1", "seed1", t0); err != nil {
		t.Fatal(err)
	}
	if err := r.Add(ctx, "k2", "seed2", t0.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	keys, err := r.List(ctx)
	if err != nil || len(keys) != 2 {
		t.Fatalf("keys = %v, %v", keys, err)
	}
	if keys[0].KID != "k1" || keys[0].Seed != "seed1" || keys[0].RetiredAt == nil ||
		!keys[0].RetiredAt.Equal(t0.Add(time.Hour)) || !keys[0].CreatedAt.Equal(t0) {
		t.Fatalf("retired key = %+v", keys[0])
	}
	if keys[1].KID != "k2" || keys[1].RetiredAt != nil {
		t.Fatalf("current key = %+v", keys[1])
	}
}
