// Package tags serves /tags of a space.
package tags

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/ledger"
	"savvy-go/internal/refsync"
)

// Handler serves the tags routes; changes go through the sync groups.
type Handler struct{ Refsync refsync.Service }

func (s Handler) tagsIndex(w http.ResponseWriter, r *http.Request) {
	list, err := httpx.Scope(r).Tags.All(r.Context())
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(list, dto.NewTag))
}

func (s Handler) tagsStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		httpx.WriteValidation(w, map[string][]string{"name": {"The name field is required."}})
		return
	}
	t, err := s.Refsync.CreateTag(r.Context(), httpx.User(r), httpx.Scope(r).ID, body.Name)
	if err != nil {
		if !httpx.WriteSyncError(w, err) {
			httpx.WriteValidation(w, map[string][]string{"name": {"The name has already been taken."}})
		}
		return
	}
	httpx.WriteData(w, http.StatusCreated, dto.NewTag(*t))
}

func (s Handler) tagsShow(w http.ResponseWriter, r *http.Request) {
	t := s.tagParam(w, r)
	if t == nil {
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewTag(*t))
}

func (s Handler) tagsUpdate(w http.ResponseWriter, r *http.Request) {
	cur := s.tagParam(w, r)
	if cur == nil {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		httpx.WriteValidation(w, map[string][]string{"name": {"The name field is required."}})
		return
	}
	t, err := s.Refsync.RenameTag(r.Context(), httpx.User(r), httpx.Scope(r).ID, cur.ID, body.Name)
	if err != nil {
		if !httpx.WriteSyncError(w, err) {
			httpx.WriteMessage(w, 422, err.Error())
		}
		return
	}
	if t == nil {
		httpx.WriteMessage(w, http.StatusNotFound, "Not found.")
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewTag(*t))
}

func (s Handler) tagsDestroy(w http.ResponseWriter, r *http.Request) {
	t := s.tagParam(w, r)
	if t == nil {
		return
	}
	// A failure on the plain path was never reported; one that comes from the
	// sync group (rights, another space) is.
	if err := s.Refsync.DeleteTag(r.Context(), httpx.User(r), httpx.Scope(r).ID, t.ID); err != nil && httpx.WriteSyncError(w, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Handler) tagParam(w http.ResponseWriter, r *http.Request) *ledger.Tag {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	t, _ := httpx.Scope(r).Tags.ByID(r.Context(), id)
	if t == nil {
		httpx.WriteMessage(w, http.StatusNotFound, "Not found.")
	}
	return t
}
