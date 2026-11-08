package transfers

import (
	"context"
	"errors"
	"fmt"

	"savvy-go/internal/spacekeys"
	"savvy-go/internal/store"
)

// EnsureKeys gives every space on this server a signing key: the converter and
// a backup from before keys leave none. Spaces made here get theirs at creation.
// A space that is unavailable is skipped and done on a later start.
func (s Service) EnsureKeys(ctx context.Context) error {
	ids, err := s.st().Spaces(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, id := range ids {
		d, err := s.st().Space(ctx, id)
		if err != nil {
			continue
		}
		if err := store.Tx(ctx, d, func(tx store.DB) error {
			_, err := spacekeys.Current(ctx, tx)
			return err
		}); err != nil {
			errs = append(errs, fmt.Errorf("space %d: %w", id, err))
		}
	}
	return errors.Join(errs...)
}

// Resign moves the transfers v1.0 left at sig_v 1 (the server's signature) to
// sig_v 2: signed over the same stored bytes by the key of each space, in both
// spaces in one step. It touches a transfer only when the server signature
// verifies and both spaces hold the same confirmed record; anything else stays
// as it is (frozen when unverifiable) until the sync or a decision settles it.
// Run it after every space is converted and EnsureKeys and SyncAll are done.
func (s Service) Resign(ctx context.Context) error {
	links, err := s.server().AllLinks(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, l := range links {
		if err := s.resignLink(ctx, l.A, l.B); err != nil {
			errs = append(errs, fmt.Errorf("link %d-%d: %w", l.A, l.B, err))
		}
	}
	return errors.Join(errs...)
}

func (s Service) resignLink(ctx context.Context, a, b int64) error {
	byID, _, err := s.spaceUUIDs(ctx)
	if err != nil {
		return err
	}
	return s.st().InSpaces(ctx, []int64{a, b}, func(dbs map[int64]store.DB) error {
		ra, err := loadTransfers(ctx, dbs[a])
		if err != nil {
			return err
		}
		rb, err := loadTransfers(ctx, dbs[b])
		if err != nil {
			return err
		}
		v, err := s.verifier(ctx, dbs)
		if err != nil {
			return err
		}
		for _, id := range unionKeys(ra, rb) {
			x, inA := ra[id]
			y, inB := rb[id]
			if !inA || !inB || x.SigV != 1 || y.SigV != 1 || x.Status != transferConfirmed ||
				y.Status != transferConfirmed || !x.same(y) || !x.involves(byID[a].UUID, byID[b].UUID) || !v.ok(x) {
				continue
			}
			if err := s.signStored(ctx, dbs, &x); err != nil {
				return err
			}
			for _, id := range []int64{a, b} {
				if err := writeRecord(ctx, dbs[id], byID[id].UUID, x, transferConfirmed, nil, nil); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
