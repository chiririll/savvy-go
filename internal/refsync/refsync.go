// Package refsync keeps the categories and tags of the spaces in a sync group
// identical. Operations on them are mirrored: what is done in one space is
// done in all of them, in one transaction, matched by name. Joining a group
// (or switching a kind on) first reconciles the spaces, see Apply.
//
// It sits on top of ledger and is independent of transfers: a sync group has
// nothing to do with the links that spaces need to send money to each other.
package refsync

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"savvy-go/internal/auth"
	repo "savvy-go/internal/db/repo/refsync"
	"savvy-go/internal/ledger"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
)

// Kind is what a group keeps identical.
type Kind string

const (
	Categories Kind = "categories"
	Tags       Kind = "tags"
)

var (
	// ErrRights is returned when a change would reach a space where the actor
	// may not edit.
	ErrRights = errors.New("you need to be an editor in every space of the sync group")
	// ErrNotAdmin is returned when a change of the group needs an admin of a space.
	ErrNotAdmin = errors.New("you need to be an admin of the spaces involved")
	// ErrUnresolved is returned when spaces differ in a way nobody decided.
	ErrUnresolved = errors.New("the spaces differ in ways that are not decided yet")
	// ErrInvalid is returned for a decision that cannot be carried out.
	ErrInvalid = errors.New("invalid decision")
	// ErrDiverged is returned when a space of the group lacks what the others
	// have, so the change cannot be mirrored there. Reconcile the group again.
	ErrDiverged = errors.New("the spaces of the group have drifted apart")
	// ErrInGroup is returned for a space that is in another group already.
	ErrInGroup = errors.New("a space can be in one sync group only")
	// ErrTooFew is returned when a group would have fewer than two spaces.
	ErrTooFew = errors.New("a sync group needs at least two spaces")
	// ErrNoGroup is returned for a group that does not exist.
	ErrNoGroup = errors.New("no such sync group")
	// ErrNotMember is returned for removing a space that is not in the group.
	ErrNotMember = errors.New("the space is not in the group")

	errStop = errors.New("nothing to do") // the object of the operation is gone
)

// MirrorError is an error from a space other than the one asked: the change
// worked there but not in this other space of the group.
type MirrorError struct {
	Space int64
	Name  string
	Err   error
}

func (e *MirrorError) Error() string { return fmt.Sprintf("space %q: %v", e.Name, e.Err) }
func (e *MirrorError) Unwrap() error { return e.Err }

// Service is the sync groups of a server.
type Service struct{ Spaces spaces.Service }

func (s Service) st() store.Store { return s.Spaces.Store }

func (s Service) server() repo.Server { return repo.Server{DB: s.st().Server()} }

// Group is a sync group.
type Group struct {
	ID         int64
	Categories bool
	Tags       bool
	Members    []int64 // ascending
}

func (g Group) syncs(k Kind) bool {
	if k == Categories {
		return g.Categories
	}
	return g.Tags
}

// GroupOf is the group a space is in; nil when it is in none, or alone.
func (s Service) GroupOf(ctx context.Context, spaceID int64) (*Group, error) {
	g, err := s.server().GroupOf(ctx, spaceID)
	if g == nil || err != nil {
		return nil, err
	}
	return s.group(ctx, g)
}

func (s Service) group(ctx context.Context, g *repo.Group) (*Group, error) {
	members, err := s.server().Members(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	if len(members) < 2 {
		return nil, nil
	}
	return &Group{ID: g.ID, Categories: g.Categories, Tags: g.Tags, Members: members}, nil
}

// requireEditor checks the actor may edit every space; a failure names the space.
func (s Service) requireEditor(ctx context.Context, actor *auth.User, ids []int64) error {
	for _, id := range ids {
		role, err := s.Spaces.Role(ctx, id, actor.ID)
		if err != nil {
			return err
		}
		if !spaces.CanWrite(role) {
			return fmt.Errorf("%w (%s)", ErrRights, s.name(ctx, id))
		}
	}
	return nil
}

func (s Service) name(ctx context.Context, id int64) string {
	if sp, err := s.Spaces.Get(ctx, id); err == nil && sp != nil {
		return sp.Name
	}
	return fmt.Sprint(id)
}

func (s Service) mirrorErr(ctx context.Context, id int64, err error) error {
	return &MirrorError{Space: id, Name: s.name(ctx, id), Err: err}
}

// mirror runs fn in the space and, if the space's group keeps kind identical,
// in the rest of the group, the asked space first so its errors come out as
// they are. A failure anywhere undoes everything.
func (s Service) mirror(ctx context.Context, actor *auth.User, spaceID int64, kind Kind, fn func(id int64, d store.DB) error) error {
	ids := []int64{spaceID}
	if g, err := s.GroupOf(ctx, spaceID); err != nil {
		return err
	} else if g != nil && g.syncs(kind) {
		ids = g.Members
	}
	if len(ids) == 1 {
		d, err := s.st().Space(ctx, spaceID)
		if err != nil {
			return err
		}
		return fn(spaceID, d)
	}
	if err := s.requireEditor(ctx, actor, ids); err != nil {
		return err
	}
	order := append([]int64{spaceID}, slices.DeleteFunc(slices.Clone(ids), func(id int64) bool { return id == spaceID })...)
	return s.st().InSpaces(ctx, ids, func(dbs map[int64]store.DB) error {
		for _, id := range order {
			if err := fn(id, dbs[id]); err != nil {
				if id != spaceID && !errors.Is(err, errStop) {
					return s.mirrorErr(ctx, id, err)
				}
				return err
			}
		}
		return nil
	})
}

func orStop(err error) error {
	if errors.Is(err, errStop) {
		return nil
	}
	return err
}

// findCategory looks a category up by what identifies it across spaces.
func findCategory(ctx context.Context, c ledger.Categories, typ, name string) (*ledger.Category, error) {
	list, err := c.All(ctx, typ)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if strings.EqualFold(list[i].Name, name) {
			return &list[i], nil
		}
	}
	return nil, nil
}

func findTag(ctx context.Context, t ledger.Tags, name string) (*ledger.Tag, error) {
	list, err := t.All(ctx)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if strings.EqualFold(list[i].Name, name) {
			return &list[i], nil
		}
	}
	return nil, nil
}

// counterpart is the category in another space that stands for c.
func counterpart(ctx context.Context, d store.DB, c ledger.Category) (*ledger.Category, error) {
	found, err := findCategory(ctx, ledger.Categories{DB: d}, c.Type, c.Name)
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, ErrDiverged
	}
	return found, nil
}

// CreateCategory creates the category in the space and its group.
func (s Service) CreateCategory(ctx context.Context, actor *auth.User, spaceID int64, c ledger.Category) (*ledger.Category, error) {
	var out *ledger.Category
	err := s.mirror(ctx, actor, spaceID, Categories, func(id int64, d store.DB) error {
		created, err := (ledger.Categories{DB: d}).Create(ctx, c)
		if id == spaceID {
			out = created
		}
		return err
	})
	return out, err
}

// UpdateCategory changes the category, matched in the rest of the group by
// the name and type it had before.
func (s Service) UpdateCategory(ctx context.Context, actor *auth.User, spaceID, categoryID int64, c ledger.Category) (*ledger.Category, error) {
	var out, before *ledger.Category
	err := s.mirror(ctx, actor, spaceID, Categories, func(id int64, d store.DB) error {
		cats := ledger.Categories{DB: d}
		target := categoryID
		if id == spaceID {
			cur, err := cats.ByID(ctx, categoryID)
			if err != nil {
				return err
			}
			if cur == nil {
				return errStop
			}
			before = cur
		} else {
			other, err := counterpart(ctx, d, *before)
			if err != nil {
				return err
			}
			target = other.ID
		}
		updated, err := cats.Update(ctx, target, c)
		if id == spaceID {
			out = updated
		}
		return err
	})
	return out, orStop(err)
}

// SetDefaultCategory makes the category the default of its type everywhere.
func (s Service) SetDefaultCategory(ctx context.Context, actor *auth.User, spaceID, categoryID int64) (*ledger.Category, error) {
	var out, before *ledger.Category
	err := s.mirror(ctx, actor, spaceID, Categories, func(id int64, d store.DB) error {
		cats := ledger.Categories{DB: d}
		target := categoryID
		if id == spaceID {
			cur, err := cats.ByID(ctx, categoryID)
			if err != nil {
				return err
			}
			if cur == nil {
				return errStop
			}
			before = cur
		} else {
			other, err := counterpart(ctx, d, *before)
			if err != nil {
				return err
			}
			target = other.ID
		}
		updated, err := cats.SetDefault(ctx, target)
		if id == spaceID {
			out = updated
		}
		return err
	})
	return out, orStop(err)
}

// DeleteCategory deletes the category everywhere. One successor, chosen in
// the asked space, takes the transactions of the category in every space of
// the group that has any.
func (s Service) DeleteCategory(ctx context.Context, actor *auth.User, spaceID, categoryID int64, successorID *int64) error {
	var before, successor *ledger.Category
	err := s.mirror(ctx, actor, spaceID, Categories, func(id int64, d store.DB) error {
		cats := ledger.Categories{DB: d}
		target, succ := categoryID, successorID
		if id == spaceID {
			cur, err := cats.ByID(ctx, categoryID)
			if err != nil {
				return err
			}
			if cur == nil {
				return errStop
			}
			before = cur
			if successorID != nil {
				if successor, err = cats.ByID(ctx, *successorID); err != nil {
					return err
				}
			}
		} else {
			other, err := counterpart(ctx, d, *before)
			if err != nil {
				return err
			}
			target, succ = other.ID, nil
			if successor != nil {
				if found, err := findCategory(ctx, cats, successor.Type, successor.Name); err != nil {
					return err
				} else if found != nil {
					succ = &found.ID
				}
			}
		}
		return cats.Delete(ctx, target, succ)
	})
	return orStop(err)
}

// CreateTag creates the tag in the space and its group.
func (s Service) CreateTag(ctx context.Context, actor *auth.User, spaceID int64, name string) (*ledger.Tag, error) {
	var out *ledger.Tag
	err := s.mirror(ctx, actor, spaceID, Tags, func(id int64, d store.DB) error {
		created, err := (ledger.Tags{DB: d}).Create(ctx, name)
		if id == spaceID {
			out = created
		}
		return err
	})
	return out, err
}

// RenameTag renames the tag, matched in the rest of the group by its old name.
func (s Service) RenameTag(ctx context.Context, actor *auth.User, spaceID, tagID int64, name string) (*ledger.Tag, error) {
	var out, before *ledger.Tag
	err := s.mirror(ctx, actor, spaceID, Tags, func(id int64, d store.DB) error {
		tags := ledger.Tags{DB: d}
		target := tagID
		if id == spaceID {
			cur, err := tags.ByID(ctx, tagID)
			if err != nil {
				return err
			}
			if cur == nil {
				return errStop
			}
			before = cur
		} else {
			other, err := findTag(ctx, tags, before.Name)
			if err != nil {
				return err
			}
			if other == nil {
				return ErrDiverged
			}
			target = other.ID
		}
		updated, err := tags.Update(ctx, target, name)
		if id == spaceID {
			out = updated
		}
		return err
	})
	return out, orStop(err)
}

// DeleteTag deletes the tag everywhere; transactions lose it.
func (s Service) DeleteTag(ctx context.Context, actor *auth.User, spaceID, tagID int64) error {
	var before *ledger.Tag
	err := s.mirror(ctx, actor, spaceID, Tags, func(id int64, d store.DB) error {
		tags := ledger.Tags{DB: d}
		target := tagID
		if id == spaceID {
			cur, err := tags.ByID(ctx, tagID)
			if err != nil {
				return err
			}
			if cur == nil {
				return errStop
			}
			before = cur
		} else {
			other, err := findTag(ctx, tags, before.Name)
			if err != nil {
				return err
			}
			if other == nil {
				return ErrDiverged
			}
			target = other.ID
		}
		return tags.Delete(ctx, target)
	})
	return orStop(err)
}
