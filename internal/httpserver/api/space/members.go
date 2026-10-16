package space

import (
	"net/http"
	"strconv"

	"savvy-go/internal/auth"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/spaces"

	"github.com/go-chi/chi/v5"
)

func (h Handler) membersIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.Spaces.Members(r.Context(), httpx.Scope(r).ID)
	if err != nil {
		httpx.WriteSpaceError(w, err)
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(list, func(m spaces.Member) map[string]any {
		return map[string]any{"userId": m.UserID, "name": m.Name, "email": m.Email, "role": m.Role, "createdAt": httpx.TimeOrNil(m.CreatedAt)}
	}))
}

// memberParam is the user named by {user}, if they belong to the space.
func (h Handler) memberParam(w http.ResponseWriter, r *http.Request) *auth.User {
	id, err := strconv.ParseInt(chi.URLParam(r, "user"), 10, 64)
	if err == nil {
		if role, _ := h.Spaces.Role(r.Context(), httpx.Scope(r).ID, id); role != "" {
			if u, _ := h.Users.ByID(r.Context(), id); u != nil {
				return u
			}
		}
	}
	httpx.WriteMessage(w, http.StatusNotFound, "Not a member of this space.")
	return nil
}

func (h Handler) membersUpdate(w http.ResponseWriter, r *http.Request) {
	u := h.memberParam(w, r)
	if u == nil {
		return
	}
	var body struct {
		Role string `json:"role"`
	}
	if !httpx.DecodeBody(w, r, &body) {
		return
	}
	if !spaces.ValidRole(body.Role) {
		httpx.WriteValidation(w, map[string][]string{"role": {"The role must be admin, editor or viewer."}})
		return
	}
	if err := h.Spaces.ChangeRole(r.Context(), httpx.Scope(r).ID, u, body.Role); err != nil {
		httpx.WriteSpaceError(w, err)
		return
	}
	h.membersIndex(w, r)
}

func (h Handler) membersDestroy(w http.ResponseWriter, r *http.Request) {
	u := h.memberParam(w, r)
	if u == nil {
		return
	}
	if err := h.Spaces.RemoveMember(r.Context(), httpx.Scope(r).ID, u.ID); err != nil {
		httpx.WriteSpaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) spaceLeave(w http.ResponseWriter, r *http.Request) {
	if err := h.Spaces.RemoveMember(r.Context(), httpx.Scope(r).ID, httpx.User(r).ID); err != nil {
		httpx.WriteSpaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
