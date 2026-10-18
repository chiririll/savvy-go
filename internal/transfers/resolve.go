package transfers

import (
	"context"
	"errors"

	"savvy-go/internal/auth"
	repo "savvy-go/internal/db/repo/transfers"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
)

// Resolve decides on a pending transfer of a restored space. Accepting takes
// the other space's state into this one (editor here). Rejecting pushes this
// space's state to the other one with a new version (editor in both); a
// record whose signature this server cannot verify is never pushed (P43).
func (s Service) Resolve(ctx context.Context, actor *auth.User, spaceID int64, id string, accept bool) error {
	d, err := s.st().Space(ctx, spaceID)
	if err != nil {
		return err
	}
	cur, err := loadTransfer(ctx, d, id)
	if err != nil {
		return err
	}
	if cur.Status != transferPending || cur.Review == nil {
		return ErrNotUnderReview
	}
	own, other, err := s.counterpart(ctx, spaceID, *cur)
	if err != nil {
		return err
	}
	if err := s.authorizeResolve(ctx, actor, *cur, own, other, accept); err != nil {
		return err
	}
	return s.st().InSpaces(ctx, []int64{own.ID, other.ID}, func(dbs map[int64]store.DB) error {
		mine := dbs[own.ID]
		cur, err := loadTransfer(ctx, mine, id)
		if err != nil {
			return err
		}
		theirs, theirsErr := loadTransfer(ctx, dbs[other.ID], id)
		if theirsErr != nil && !errors.Is(theirsErr, ErrGone) {
			return theirsErr
		}
		if accept {
			return s.acceptReview(ctx, mine, own, id, *cur, theirs)
		}
		return s.rejectReview(ctx, dbs, own, other, *cur, theirs)
	})
}

// authorizeResolve checks the actor may decide: editor in this space to accept,
// in both to reject, and a reject only pushes a record this server can verify.
func (s Service) authorizeResolve(
	ctx context.Context, actor *auth.User, cur Transfer, own, other spaces.Space, accept bool,
) error {
	if accept {
		return s.requireEditor(ctx, actor, own.ID)
	}
	if err := s.requireEditor(ctx, actor, own.ID, other.ID); err != nil {
		return err
	}
	if *cur.Review != reviewCreatedRemote && !verifyTransfer(s.Keys, cur) {
		return ErrUnverified
	}
	return nil
}

// acceptReview takes the other space's state into this one.
func (s Service) acceptReview(
	ctx context.Context, mine store.DB, own spaces.Space, id string, cur Transfer, theirs *Transfer,
) error {
	switch *cur.Review {
	case reviewCreatedRemote:
		cur.Review, cur.Remote = nil, nil
		return writeRecord(ctx, mine, own.UUID, cur, "confirmed", nil, nil)
	case reviewMissingRemote:
		return repo.Space{DB: mine}.Delete(ctx, id)
	default:
		if theirs == nil {
			return ErrGone
		}
		return writeRecord(ctx, mine, own.UUID, *theirs, "confirmed", nil, nil)
	}
}

// rejectReview pushes this space's state, with a new version, to both spaces.
func (s Service) rejectReview(
	ctx context.Context, dbs map[int64]store.DB, own, other spaces.Space, cur Transfer, theirs *Transfer,
) error {
	next := cur
	next.Review, next.Remote = nil, nil
	version := cur.Version
	if theirs != nil && theirs.Version > version {
		version = theirs.Version
	}
	next.Version = version + 1
	if *cur.Review == reviewCreatedRemote {
		at := now()
		next.DeletedAt = &at
	}
	signTransfer(s.Keys, &next)
	for _, sp := range []spaces.Space{own, other} {
		if err := writeRecord(ctx, dbs[sp.ID], sp.UUID, next, "confirmed", nil, nil); err != nil {
			return err
		}
	}
	return nil
}

// Trust re-signs a transfer this server cannot verify (made on a server whose
// key it does not trust), after its members agree (P46): editor in every
// space of it that is on this server.
func (s Service) Trust(ctx context.Context, actor *auth.User, spaceID int64, id string) error {
	d, err := s.st().Space(ctx, spaceID)
	if err != nil {
		return err
	}
	cur, err := loadTransfer(ctx, d, id)
	if err != nil {
		return err
	}
	byID, byUUID, err := s.spaceUUIDs(ctx)
	if err != nil {
		return err
	}
	spaces := []int64{spaceID}
	for _, u := range []string{cur.From.SpaceUUID, cur.To.SpaceUUID} {
		if sp, ok := byUUID[u]; ok && sp.ID != spaceID {
			spaces = append(spaces, sp.ID)
		}
	}
	if err := s.requireEditor(ctx, actor, spaces...); err != nil {
		return err
	}
	return s.st().InSpaces(ctx, spaces, func(dbs map[int64]store.DB) error {
		next, err := loadTransfer(ctx, dbs[spaceID], id)
		if err != nil {
			return err
		}
		signTransfer(s.Keys, next)
		for _, sid := range spaces {
			if err := writeRecord(ctx, dbs[sid], byID[sid].UUID, *next, "confirmed", next.Review, next.Remote); err != nil {
				return err
			}
		}
		return nil
	})
}
