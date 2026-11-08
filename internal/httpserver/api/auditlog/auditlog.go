// Package auditlog serves /audit-log of a space: the change log, and rolling
// back what a token did.
package auditlog

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/audit"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/ledger"
	"savvy-go/internal/spaces"
)

const (
	defaultLimit = 50
	maxLimit     = 200
)

// entry is a row of the log as the API shows it. Snapshots are left out of a
// list and shown for one entry.
type entry struct {
	ID           int64           `json:"id"`
	At           time.Time       `json:"at"`
	Source       string          `json:"source"`
	UserID       *int64          `json:"userId"`
	TokenID      *int64          `json:"tokenId"`
	TokenLabel   string          `json:"tokenLabel,omitempty"`
	EntityType   string          `json:"entityType"`
	EntityID     *int64          `json:"entityId"`
	Op           string          `json:"op"`
	BatchID      string          `json:"batchId,omitempty"`
	RolledBackBy *int64          `json:"rolledBackBy"`
	Details      string          `json:"details,omitempty"`
	Rollbackable bool            `json:"rollbackable"`
	Blocked      string          `json:"rollbackBlockedReason,omitempty"`
	Before       json.RawMessage `json:"before,omitempty"`
	After        json.RawMessage `json:"after,omitempty"`
}

func isAdmin(r *http.Request) bool { return httpx.Scope(r).Role == spaces.RoleAdmin }

// who is the token id the caller's rollbacks are limited to: its own, or 0 for
// a space admin in the browser. ok is false for a caller who may not roll back.
func who(r *http.Request) (tokenID int64, ok bool) {
	if t := httpx.Token(r); t != nil {
		return t.ID, true
	}
	return 0, isAdmin(r)
}

// view turns a row into an entry. undo is what Undoable said about it for the
// caller; nil when the caller may not roll back at all (canUndo false).
func view(e audit.Entry, undo error, canUndo, withSnapshots bool) (entry, error) {
	out := entry{
		ID: e.ID, At: e.At, Source: e.Source, UserID: e.UserID, TokenID: e.TokenID, TokenLabel: e.TokenLabel,
		EntityType: e.EntityType, EntityID: e.EntityID, Op: e.Op, BatchID: e.BatchID, RolledBackBy: e.RolledBackBy,
		Details: e.Details,
	}
	if canUndo {
		switch {
		case undo == nil:
			out.Rollbackable = true
		case errors.Is(undo, audit.ErrNotFound):
		default:
			var ov *audit.OverlapError
			if !errors.As(undo, &ov) && !errors.Is(undo, audit.ErrAlreadyRolledBack) && !errors.Is(undo, audit.ErrNotUndoable) {
				return out, undo
			}
			out.Blocked = undo.Error()
		}
	}
	if withSnapshots {
		out.Before, out.After = e.Before, e.After
	}
	return out, nil
}

// views turns rows into entries, asking for whether each could be rolled back
// in one round for the whole page.
func views(r *http.Request, rows []audit.Entry, withSnapshots bool) ([]entry, error) {
	tok, can := who(r)
	var undo []error
	if can {
		var err error
		if undo, err = httpx.Scope(r).Rollbacks.UndoableAll(r.Context(), rows, tok); err != nil {
			return nil, err
		}
	}
	out := make([]entry, len(rows))
	for i, e := range rows {
		var u error
		if can {
			u = undo[i]
		}
		var err error
		if out[i], err = view(e, u, can, withSnapshots); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// list shows the log, newest first. A token sees its own entries, a space
// admin everything, any other member their own.
func (Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := audit.Filter{EntityType: q.Get("entity_type")}
	f.EntityID, _ = strconv.ParseInt(q.Get("entity_id"), 10, 64)
	f.BeforeID, _ = strconv.ParseInt(q.Get("before"), 10, 64)
	switch t := httpx.Token(r); {
	case t != nil:
		f.TokenID = t.ID
	case isAdmin(r):
		f.TokenID, _ = strconv.ParseInt(q.Get("token_id"), 10, 64)
	default:
		f.UserID = httpx.User(r).ID
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > maxLimit {
		limit = defaultLimit
	}
	rows, err := httpx.Scope(r).Rollbacks.Log().List(r.Context(), f, limit)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	out, err := views(r, rows, false)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, out)
}

func (Handler) show(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	e, err := httpx.Scope(r).Rollbacks.Log().Get(r.Context(), id)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if e == nil || !mayRead(r, *e) {
		httpx.WriteMessage(w, http.StatusNotFound, "No such entry.")
		return
	}
	one, err := views(r, []audit.Entry{*e}, true)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, one[0])
}

// mayRead is whether the caller may see e: see list.
func mayRead(r *http.Request, e audit.Entry) bool {
	switch t := httpx.Token(r); {
	case t != nil:
		return e.TokenID != nil && *e.TokenID == t.ID
	case isAdmin(r):
		return true
	}
	return e.UserID != nil && *e.UserID == httpx.User(r).ID
}

type rollbackBody struct {
	IDs     []int64 `json:"ids"`
	BatchID string  `json:"batch_id"`
}

// rollback undoes entries, all or none. The body names either ids or a batch.
func (Handler) rollback(w http.ResponseWriter, r *http.Request) {
	tok, can := who(r)
	if !can {
		httpx.WriteMessage(w, http.StatusForbidden, "Only a space admin can roll back changes.")
		return
	}
	var body rollbackBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || (len(body.IDs) == 0) == (body.BatchID == "") {
		httpx.WriteValidation(w, map[string][]string{"ids": {"Send either ids or batch_id."}})
		return
	}
	var (
		done []int64
		err  error
	)
	if body.BatchID != "" {
		done, err = httpx.Scope(r).Rollbacks.UndoBatch(r.Context(), body.BatchID, tok)
	} else {
		done, err = httpx.Scope(r).Rollbacks.Undo(r.Context(), body.IDs, tok)
	}
	if err != nil {
		writeRollbackError(w, err)
		return
	}
	httpx.WriteData(w, http.StatusOK, map[string]any{"undone": len(done), "rollbackIds": done})
}

func writeRollbackError(w http.ResponseWriter, err error) {
	var (
		ov  *audit.OverlapError
		dup *ledger.ExternalIDExistsError
	)
	switch {
	case errors.Is(err, audit.ErrNotFound):
		httpx.WriteCoded(w, http.StatusNotFound, err.Error(), "not_found")
	case errors.As(err, &ov):
		httpx.WriteCoded(w, http.StatusConflict, err.Error(), "changed_since")
	case errors.Is(err, audit.ErrAlreadyRolledBack):
		httpx.WriteCoded(w, http.StatusConflict, err.Error(), "already_rolled_back")
	case errors.As(err, &dup):
		httpx.WriteCoded(w, http.StatusConflict, err.Error(), "external_id_exists")
	case errors.Is(err, audit.ErrNotUndoable):
		httpx.WriteCoded(w, http.StatusUnprocessableEntity, err.Error(), "not_rollbackable")
	default:
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, err.Error())
	}
}

// clear removes the log (one token's entries with ?token_id=) and leaves a
// single entry saying who did it.
func (Handler) clear(w http.ResponseWriter, r *http.Request) {
	tokenID, _ := strconv.ParseInt(r.URL.Query().Get("token_id"), 10, 64)
	n, err := httpx.Scope(r).Rollbacks.Log().Clear(r.Context(), tokenID)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, map[string]any{"removed": n})
}
