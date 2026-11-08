// Package audit is the space's change log: who changed what, through which
// door, with enough of the entity before and after to undo it.
//
// It knows no entity. A service writes one row per entity it changes, in the
// database transaction of the change, and supplies the inverse of its own
// rows to Undo. Undo holds the rules every entity shares: a token undoes only
// its own rows, and only while nothing has changed the entity since.
package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	repo "savvy-go/internal/db/repo/audit"
	"savvy-go/internal/store"
)

// The doors a change comes through. The server decides, never the client.
const (
	SourceUI     = "ui"     // a signed-in browser session
	SourceAPI    = "api"    // a personal API token
	SourceMCP    = "mcp"    // an MCP client's token
	SourceSystem = "system" // no request: automation, recurring rules, imports, cleanup
)

// What a row records.
const (
	OpCreate   = "create"
	OpUpdate   = "update"
	OpDelete   = "delete"
	OpConfirm  = "confirm"
	OpSkip     = "skip"
	OpRollback = "rollback"
	OpCleanup  = "cleanup"
)

// Entity types of the log itself.
const EntityAudit = "audit"

// Entry is one row of the log.
type Entry = repo.Row

// Filter narrows a listing; a zero field is no filter.
type Filter = repo.Filter

// Actor is who a change is attributed to. The zero Actor is the system: no
// user, no token. A user acting through the browser has no token.
type Actor struct {
	UserID     *int64
	TokenID    *int64
	TokenLabel string
	Source     string
}

type ctxKey struct{}

// WithActor returns ctx carrying the actor of a request.
func WithActor(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, ctxKey{}, a)
}

// ActorFrom is the actor of ctx, the system when there is none.
func ActorFrom(ctx context.Context) Actor {
	a, _ := ctx.Value(ctxKey{}).(Actor)
	if a.Source == "" {
		a.Source = SourceSystem
	}
	return a
}

// Change is a row to write. Before and After are encoded with Snapshot. An
// EntityID of 0 is no entity: the row is about a whole operation (a CSV
// import) and cannot be rolled back by entity.
type Change struct {
	EntityType string
	EntityID   int64
	Op         string
	Before     json.RawMessage
	After      json.RawMessage
	BatchID    string
	Details    string
}

// Snapshot encodes v for Before or After; nil (also a nil pointer) is none.
func Snapshot(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Pointer && rv.IsNil() {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}

// Log is the audit log of one space.
type Log struct {
	DB  store.DB
	Now func() time.Time // the clock; time.Now when nil
}

func (l Log) repo() repo.Repo { return repo.Repo{DB: l.DB} }

func (l Log) now() time.Time {
	if l.Now != nil {
		return l.Now().UTC()
	}
	return time.Now().UTC()
}

// Record writes a row on d, attributed to the actor of ctx, and returns its
// id. Call it with the handle of the transaction that makes the change.
func (l Log) Record(ctx context.Context, d store.DB, c Change) (int64, error) {
	a := ActorFrom(ctx)
	id := c.EntityID
	var entity *int64
	if c.EntityType != EntityAudit && id != 0 {
		entity = &id
	}
	return repo.Repo{DB: d}.Insert(ctx, repo.New{
		At: l.now(), Source: a.Source, UserID: a.UserID, TokenID: a.TokenID, TokenLabel: a.TokenLabel,
		EntityType: c.EntityType, EntityID: entity, Op: c.Op, Before: c.Before, After: c.After,
		BatchID: c.BatchID, Details: c.Details,
	})
}

// Get returns nil for an unknown id.
func (l Log) Get(ctx context.Context, id int64) (*Entry, error) { return l.repo().Get(ctx, id) }

// List returns up to limit entries, newest first.
func (l Log) List(ctx context.Context, f Filter, limit int) ([]Entry, error) {
	return l.repo().List(ctx, f, limit)
}

// ErrNotFound is the answer for an entry that does not exist or is not the
// caller's to undo; the two look the same on purpose.
var ErrNotFound = errors.New("no such entry in the audit log")

// ErrAlreadyRolledBack is the answer for an entry that was undone before.
var ErrAlreadyRolledBack = errors.New("this change was already rolled back")

// ErrNotUndoable wraps the reason an entry cannot be undone at all.
var ErrNotUndoable = errors.New("this change cannot be rolled back")

// OverlapError says which later change stands in the way of an undo.
type OverlapError struct {
	Entry, By  int64
	EntityType string
	EntityID   int64
}

func (e *OverlapError) Error() string {
	return fmt.Sprintf("%s %d was changed again by audit entry %d; roll that back first, or fix it by hand",
		e.EntityType, e.EntityID, e.By)
}

// Undoable says whether e could be undone now by the holder of tokenID (0:
// anyone entitled to, as a space admin in the browser), and why not. The
// support check is the entity service's: whether it knows an inverse.
func (l Log) Undoable(ctx context.Context, e Entry, tokenID int64, supported func(Entry) error) error {
	return undoable(ctx, l.repo(), e, tokenID, supported)
}

func undoable(ctx context.Context, r repo.Repo, e Entry, tokenID int64, supported func(Entry) error) error {
	if err := undoableAlone(e, tokenID, supported); err != nil {
		return err
	}
	by, err := r.FirstOverlap(ctx, e.EntityType, *e.EntityID, e.ID)
	if err != nil {
		return err
	}
	return overlap(e, by)
}

func overlap(e Entry, by int64) error {
	if by != 0 {
		return &OverlapError{Entry: e.ID, By: by, EntityType: e.EntityType, EntityID: *e.EntityID}
	}
	return nil
}

// UndoableAll is Undoable for many entries with one query per entity type
// instead of one per entry; errs[i] answers es[i].
func (l Log) UndoableAll(ctx context.Context, es []Entry, tokenID int64, supported func(Entry) error) ([]error, error) {
	errs := make([]error, len(es))
	idsOf := map[string][]int64{}
	oldest := map[string]int64{}
	for i, e := range es {
		if errs[i] = undoableAlone(e, tokenID, supported); errs[i] != nil {
			continue
		}
		t := e.EntityType
		idsOf[t] = append(idsOf[t], *e.EntityID)
		if o, ok := oldest[t]; !ok || e.ID < o {
			oldest[t] = e.ID
		}
	}
	later := map[string][]repo.Standing{}
	for t, ids := range idsOf {
		rows, err := l.repo().StandingAfter(ctx, t, ids, oldest[t])
		if err != nil {
			return nil, err
		}
		later[t] = rows
	}
	for i, e := range es {
		if errs[i] != nil {
			continue
		}
		for _, s := range later[e.EntityType] { // oldest first
			if s.EntityID == *e.EntityID && s.ID > e.ID {
				errs[i] = overlap(e, s.ID)
				break
			}
		}
	}
	return errs, nil
}

// undoableAlone is what Undoable decides from the entry alone.
func undoableAlone(e Entry, tokenID int64, supported func(Entry) error) error {
	if tokenID != 0 && (e.TokenID == nil || *e.TokenID != tokenID) {
		return ErrNotFound
	}
	if e.RolledBackBy != nil {
		return ErrAlreadyRolledBack
	}
	if err := supported(e); err != nil {
		return fmt.Errorf("%w: %w", ErrNotUndoable, err)
	}
	if e.EntityID == nil {
		return fmt.Errorf("%w: it names no entity", ErrNotUndoable)
	}
	return nil
}

// Inverse applies the inverse of e on d and returns the entity's snapshots
// for the rollback row (before: how it stood, after: how it stands now).
type Inverse func(ctx context.Context, d store.DB, e Entry) (before, after json.RawMessage, err error)

// Undo rolls back the entries ids (and nothing else), newest first, all or
// none, in one database transaction. tokenID restricts it to that token's own
// entries; 0 lifts that for a caller entitled to undo anyone's. Each entry
// must stand and no later change may overlap it: the entries of one call do
// not count against each other, since the newer ones are undone first. Every
// undone entry gets a rollback row, attributed to the actor of ctx.
func (l Log) Undo(
	ctx context.Context, ids []int64, tokenID int64, supported func(Entry) error, inverse Inverse,
) ([]int64, error) {
	var rolled []int64
	err := store.Tx(ctx, l.DB, func(d store.DB) error {
		rolled = rolled[:0]
		r := repo.Repo{DB: d}
		entries := make([]Entry, 0, len(ids))
		for _, id := range ids {
			e, err := r.Get(ctx, id)
			if err != nil {
				return err
			}
			if e == nil {
				return ErrNotFound
			}
			entries = append(entries, *e)
		}
		sortNewestFirst(entries)
		for _, e := range entries {
			if err := undoable(ctx, r, e, tokenID, supported); err != nil {
				return err
			}
			before, after, err := inverse(ctx, d, e)
			if err != nil {
				return err
			}
			by, err := l.Record(ctx, d, Change{
				EntityType: e.EntityType, EntityID: *e.EntityID, Op: OpRollback, Before: before, After: after,
				BatchID: e.BatchID, Details: fmt.Sprintf("undoes entry %d (%s)", e.ID, e.Op),
			})
			if err != nil {
				return err
			}
			if ok, err := r.MarkRolledBack(ctx, e.ID, by); err != nil || !ok {
				if err == nil {
					err = ErrAlreadyRolledBack
				}
				return err
			}
			rolled = append(rolled, by)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rolled, nil
}

func sortNewestFirst(es []Entry) {
	for i := 1; i < len(es); i++ {
		for j := i; j > 0 && es[j].ID > es[j-1].ID; j-- {
			es[j], es[j-1] = es[j-1], es[j]
		}
	}
}

// Batch lists the entries of one bulk call that still stand, newest first.
func (l Log) Batch(ctx context.Context, batchID string) ([]Entry, error) {
	all, err := l.repo().Batch(ctx, batchID)
	if err != nil {
		return nil, err
	}
	out := all[:0]
	for _, e := range all {
		if e.Op != OpRollback && e.RolledBackBy == nil {
			out = append(out, e)
		}
	}
	return out, nil
}

// ErrNeedsUser is the answer for a cleanup with no user behind it.
var ErrNeedsUser = errors.New("only a signed-in space admin can clear the audit log")

// Clear removes every entry that exists now (those of one token when tokenID
// is not 0) and writes one entry saying who did it, when and how many rows
// went. That entry is the first of the new log. Nothing else deletes entries.
// The caller has checked that the user is a space admin.
func (l Log) Clear(ctx context.Context, tokenID int64) (int64, error) {
	a := ActorFrom(ctx)
	if a.UserID == nil {
		return 0, ErrNeedsUser
	}
	var removed int64
	err := store.Tx(ctx, l.DB, func(d store.DB) error {
		r := repo.Repo{DB: d}
		max, err := r.MaxID(ctx)
		if err != nil {
			return err
		}
		if removed, err = r.DeleteUpTo(ctx, max, tokenID); err != nil {
			return err
		}
		scope := "all entries"
		if tokenID != 0 {
			scope = fmt.Sprintf("entries of token %d", tokenID)
		}
		_, err = l.Record(ctx, d, Change{
			EntityType: EntityAudit, Op: OpCleanup, Details: fmt.Sprintf("removed %d %s", removed, scope),
		})
		return err
	})
	return removed, err
}
