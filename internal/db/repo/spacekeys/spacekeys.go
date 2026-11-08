// Package spacekeys (repo) stores the signing keys of one space in its database.
package spacekeys

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

// Repo reads and writes the keys of one space database.
type Repo struct{ DB store.DB }

func (r Repo) q() *sqlc.Queries { return db.Q(r.DB) }

// Key is one signing key of a space. Seed is the private key, base64: it stays
// in the space file and leaves it only with the space's backups.
type Key struct {
	KID, Seed string
	CreatedAt time.Time
	RetiredAt *time.Time
}

// List returns the keys, oldest first. At most one is current (RetiredAt nil).
func (r Repo) List(ctx context.Context) ([]Key, error) {
	rows, err := r.q().ListSpaceKeys(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Key, len(rows))
	for i, row := range rows {
		out[i] = Key{KID: row.Kid, Seed: row.Seed}
		out[i].CreatedAt, _ = db.ParseTimeString(row.CreatedAt)
		if at, ok := db.ParseTime(row.RetiredAt); ok {
			out[i].RetiredAt = &at
		}
	}
	return out, nil
}

// Add makes a key current and retires the one before it.
func (r Repo) Add(ctx context.Context, kid, seed string, now time.Time) error {
	if err := r.q().RetireSpaceKeys(ctx, db.TimeNS(now)); err != nil {
		return err
	}
	return r.q().InsertSpaceKey(ctx, sqlc.InsertSpaceKeyParams{Kid: kid, Seed: seed, CreatedAt: db.FormatTime(now)})
}
