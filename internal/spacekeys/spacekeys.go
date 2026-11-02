// Package spacekeys gives every space its own Ed25519 key. A transfer between
// two spaces is signed by both keys (internal/transfers); the keys live in the
// space files and travel with their backups, unlike the server key.
package spacekeys

import (
	"context"
	"crypto/ed25519"
	"time"

	repo "savvy-go/internal/db/repo/spacekeys"
	"savvy-go/internal/signing"
	"savvy-go/internal/store"
)

// Current returns the current key of the space whose database is d, making
// one when the space has none (a new space, a converted v1.0 file, a backup
// from before keys). d should be the handle of the transaction the caller
// writes in.
func Current(ctx context.Context, d store.DB) (*signing.Key, error) {
	r := repo.Repo{DB: d}
	keys, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, k := range keys {
		if k.RetiredAt == nil {
			return signing.Parse([]byte(k.Seed))
		}
	}
	return create(ctx, r)
}

// Rotate retires the current key and makes a new one. Retired keys still
// verify what they signed.
func Rotate(ctx context.Context, d store.DB) (*signing.Key, error) {
	return create(ctx, repo.Repo{DB: d})
}

func create(ctx context.Context, r repo.Repo) (*signing.Key, error) {
	key, err := signing.New()
	if err != nil {
		return nil, err
	}
	if err := r.Add(ctx, key.KID(), key.SeedString(), time.Now().UTC()); err != nil {
		return nil, err
	}
	return key, nil
}

// Public lists the public keys of the space by kid, retired ones included.
func Public(ctx context.Context, d store.DB) (map[string]ed25519.PublicKey, error) {
	keys, err := (repo.Repo{DB: d}).List(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]ed25519.PublicKey, len(keys))
	for _, k := range keys {
		key, err := signing.Parse([]byte(k.Seed))
		if err != nil {
			continue // a damaged key verifies nothing
		}
		out[k.KID] = key.Public()
	}
	return out, nil
}
