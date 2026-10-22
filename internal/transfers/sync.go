package transfers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"

	repo "savvy-go/internal/db/repo/transfers"
	"savvy-go/internal/store"
)

// winner picks between two different versions of one record: the higher
// version, and between equal versions the one with the greater content hash,
// so both sides choose alike (P23).
func winner(a, b Transfer) Transfer {
	if a.Version != b.Version {
		if a.Version > b.Version {
			return a
		}
		return b
	}
	pa, _ := a.payload(1)
	pb, _ := b.payload(1)
	ha, hb := sha256.Sum256(pa), sha256.Sum256(pb)
	if bytes.Compare(ha[:], hb[:]) >= 0 {
		return a
	}
	return b
}

// sync merges the transfers of two spaces. With review set to one of them
// (a space just restored) differences become pending records in that space
// only; otherwise the newer verified version is copied to the side behind.
func (s Service) sync(ctx context.Context, a, b, review int64) error {
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
		sa := syncSide{id: a, uuid: byID[a].UUID, db: dbs[a]}
		sb := syncSide{id: b, uuid: byID[b].UUID, db: dbs[b]}
		for _, id := range unionKeys(ra, rb) {
			x, inA := ra[id]
			y, inB := rb[id]
			sa.rec, sa.has, sb.rec, sb.has = x, inA, y, inB
			if err := s.syncOne(ctx, sa, sb, review); err != nil {
				return err
			}
		}
		return nil
	})
}

// syncSide is one space of a sync with its copy of the transfer at hand.
type syncSide struct {
	id   int64
	uuid string
	db   store.DB
	rec  Transfer
	has  bool
}

func unionKeys(a, b map[string]Transfer) []string {
	ids := map[string]bool{}
	for id := range a {
		ids[id] = true
	}
	for id := range b {
		ids[id] = true
	}
	keys := make([]string, 0, len(ids))
	for id := range ids {
		keys = append(keys, id)
	}
	slices.Sort(keys)
	return keys
}

func (s Service) syncOne(ctx context.Context, a, b syncSide, review int64) error {
	rec := a.rec
	if !a.has {
		rec = b.rec
	}
	if !rec.involves(a.uuid, b.uuid) {
		return nil
	}
	if review != 0 {
		own, other := a, b
		if review == b.id {
			own, other = b, a
		}
		return s.reviewOne(ctx, own.db, own.uuid, own.rec, other.rec, own.has, other.has)
	}
	if (a.has && a.rec.Status == transferPending) || (b.has && b.rec.Status == transferPending) {
		return nil // waits for a decision; never moved automatically
	}
	switch {
	case a.has && !b.has:
		return s.copyVerified(ctx, b, a.rec)
	case b.has && !a.has:
		return s.copyVerified(ctx, a, b.rec)
	case !a.rec.same(b.rec):
		w := winner(a.rec, b.rec)
		target := b
		if w.same(b.rec) {
			target = a
		}
		return s.copyVerified(ctx, target, w)
	}
	return nil
}

// copyVerified writes rec into the side behind, if its signature holds.
func (s Service) copyVerified(ctx context.Context, to syncSide, rec Transfer) error {
	if !verifyTransfer(s.Keys, rec) {
		return nil
	}
	return writeRecord(ctx, to.db, to.uuid, rec, "confirmed", nil, nil)
}

// reviewOne records, in a restored space, how its copy of a transfer differs
// from the other space's. Nothing is written to the other space.
func (s Service) reviewOne(
	ctx context.Context,
	own store.DB,
	ownUUID string,
	mine, theirs Transfer,
	haveMine, haveTheirs bool,
) error {
	mark := func(t Transfer, kind string, remote *Transfer, local string) error {
		k := kind
		return writeRecord(ctx, own, ownUUID, t, local, &k, remote)
	}
	switch {
	case !haveMine && haveTheirs:
		if theirs.DeletedAt != nil || !verifyTransfer(s.Keys, theirs) {
			return nil
		}
		// Created after the backup: here as pending, outside the balance.
		return mark(theirs, reviewCreatedRemote, nil, "pending")
	case haveMine && !haveTheirs:
		if mine.DeletedAt != nil {
			return nil
		}
		return mark(mine, reviewMissingRemote, nil, "confirmed")
	case mine.same(theirs) && mine.Version == theirs.Version:
		return nil
	case theirs.DeletedAt != nil && mine.DeletedAt == nil:
		if !verifyTransfer(s.Keys, theirs) {
			return nil
		}
		r := theirs
		return mark(mine, reviewDeletedRemote, &r, "confirmed")
	default:
		if !verifyTransfer(s.Keys, theirs) {
			return nil
		}
		r := theirs
		if mine.DeletedAt != nil {
			// Deleted in the backup, alive there: the record is kept as the
			// tombstone; the other version waits as remote.
			return writeRecord(ctx, own, ownUUID, mine, "confirmed", ptr(reviewChangedRemote), &r)
		}
		return mark(mine, reviewChangedRemote, &r, "confirmed")
	}
}

// SyncAll merges every link, as at startup. A space marked as restored is
// merged in review mode, and its mark cleared once all its links are done.
func (s Service) SyncAll(ctx context.Context) error {
	links, err := s.server().AllLinks(ctx)
	if err != nil {
		return err
	}
	marked := map[int64]bool{}
	failed := map[int64]bool{}
	for _, l := range links {
		for _, id := range []int64{l.A, l.B} {
			if _, seen := marked[id]; !seen {
				marked[id] = s.reviewPending(ctx, id)
			}
		}
	}
	var errs []error
	for _, l := range links {
		review := int64(0)
		switch {
		case marked[l.A]:
			review = l.A
		case marked[l.B]:
			review = l.B
		}
		if err := s.sync(ctx, l.A, l.B, review); err != nil {
			errs = append(errs, fmt.Errorf("link %d-%d: %w", l.A, l.B, err))
			failed[l.A], failed[l.B] = true, true
		}
	}
	for id, m := range marked {
		if m && !failed[id] {
			if err := s.clearReview(ctx, id); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func (s Service) reviewPending(ctx context.Context, id int64) bool {
	d, err := s.st().Space(ctx, id)
	if err != nil {
		return false
	}
	pending, err := repo.Space{DB: d}.ReviewPending(ctx)
	return err == nil && pending
}

func (s Service) clearReview(ctx context.Context, id int64) error {
	d, err := s.st().Space(ctx, id)
	if err != nil {
		return err
	}
	return repo.Space{DB: d}.ClearReview(ctx)
}

// AfterRestore merges a restored space with every space it is linked to, in
// review mode (P24); the review mark written before the restore went live is
// cleared when all succeeded, so a crash midway reviews again at startup.
// Records that do not involve this space at all are dropped first (P51).
func (s Service) AfterRestore(ctx context.Context, spaceID int64) error {
	links, err := s.Links(ctx, spaceID)
	if err != nil {
		return err
	}
	if err := s.dropForeign(ctx, spaceID); err != nil {
		return err
	}
	var errs []error
	for _, other := range links {
		if err := s.sync(ctx, spaceID, other.ID, spaceID); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return s.clearReview(ctx, spaceID)
}

// dropForeign removes transfer records that do not involve the space at all:
// genuine records of other spaces slipped into its backup (P51).
func (s Service) dropForeign(ctx context.Context, spaceID int64) error {
	byID, _, err := s.spaceUUIDs(ctx)
	if err != nil {
		return err
	}
	own := byID[spaceID].UUID
	d, err := s.st().Space(ctx, spaceID)
	if err != nil {
		return err
	}
	return store.Tx(ctx, d, func(tx store.DB) error {
		all, err := loadTransfers(ctx, tx)
		if err != nil {
			return err
		}
		for id, t := range all {
			if _, _, ok := t.side(own); !ok {
				if err := (repo.Space{DB: tx}).Delete(ctx, id); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
