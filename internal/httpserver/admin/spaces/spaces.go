// Package spaces serves the server admin's view of spaces: every space, deleted ones
// and their restore, who administers them, and the invitation cooldown.
package spaces

import (
	"net/http"
	"strconv"

	"savvy-go/internal/auth"
	"savvy-go/internal/backup"
	"savvy-go/internal/httpserver/httpx"
	backupx "savvy-go/internal/httpserver/httpx/backups"
	"savvy-go/internal/httpserver/httpx/dto"
	core "savvy-go/internal/spaces"

	"github.com/go-chi/chi/v5"
)

// Handler holds what the space admin routes need.
type Handler struct {
	Spaces  core.Service
	Users   auth.Users
	Backups backup.Service
}

// clearedBody says how many invitation cooldowns were lifted.
type clearedBody struct {
	Cleared int `json:"cleared"`
}

// adminSpacesIndex is the overview of every space: metadata only (P5).
func (h Handler) adminSpacesIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.Spaces.Overview(r.Context())
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(list, dto.NewSpaceOverview))
}

func (h Handler) adminSpaceParam(w http.ResponseWriter, r *http.Request) *core.Space {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err == nil {
		if space, _ := h.Spaces.Get(r.Context(), id); space != nil {
			return space
		}
	}
	httpx.WriteMessage(w, http.StatusNotFound, "Space not found.")
	return nil
}

func (h Handler) adminSpaceUpdate(w http.ResponseWriter, r *http.Request) {
	space := h.adminSpaceParam(w, r)
	if space == nil {
		return
	}
	var body struct {
		QuotaBytes *int64 `json:"quota_bytes"`
	}
	if !httpx.DecodeBody(w, r, &body) {
		return
	}
	var bytes int64
	if body.QuotaBytes != nil {
		bytes = *body.QuotaBytes
	}
	if err := h.Spaces.SetQuota(r.Context(), httpx.User(r), space.ID, bytes); err != nil {
		httpx.WriteSpaceError(w, err)
		return
	}
	h.adminSpacesIndex(w, r)
}

func (h Handler) adminSpaceDestroy(w http.ResponseWriter, r *http.Request) {
	space := h.adminSpaceParam(w, r)
	if space == nil {
		return
	}
	if err := h.Spaces.Delete(r.Context(), *space, h.Backups); err != nil {
		httpx.WriteSpaceError(w, err)
		return
	}
	_ = h.Spaces.Audit(r.Context(), httpx.User(r), "delete_space", &space.ID, nil, space.Name)
	w.WriteHeader(http.StatusNoContent)
}

// adminSpaceAssignAdmin makes a user (the caller included) admin of a space,
// outside its admins' control; the space's admins see it in their audit log.
func (h Handler) adminSpaceAssignAdmin(w http.ResponseWriter, r *http.Request) {
	space := h.adminSpaceParam(w, r)
	if space == nil {
		return
	}
	var body struct {
		UserID int64 `json:"user_id"`
	}
	if !httpx.DecodeBody(w, r, &body) {
		return
	}
	u, _ := h.Users.ByID(r.Context(), body.UserID)
	if u == nil || u.Deleted {
		httpx.WriteMessage(w, http.StatusNotFound, "User not found.")
		return
	}
	if err := h.Spaces.AssignAdmin(r.Context(), httpx.User(r), space.ID, u); err != nil {
		httpx.WriteSpaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) adminDeletedIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.Backups.DeletedBackups()
	backupx.List(w, list, err)
}

// adminDeletedRestore brings a deleted space back as a new space owned by the
// calling server admin.
func (h Handler) adminDeletedRestore(w http.ResponseWriter, r *http.Request) {
	path, err := h.Backups.DeletedPath(chi.URLParam(r, "name"))
	if err != nil {
		backupx.WriteError(w, err)
		return
	}
	space, err := h.Backups.ImportSpace(r.Context(), h.Spaces, path, "", httpx.User(r), 0)
	if err != nil {
		backupx.WriteError(w, err)
		return
	}
	_ = h.Spaces.Audit(r.Context(), httpx.User(r), "restore_deleted_space", &space.ID, nil, space.Name)
	httpx.WriteData(w, http.StatusCreated, dto.NewSpace(*space))
}

// invitationsClear ends the cooldown of revoked and expired invitations, for
// one address or all of them. It is for server admins only.
func (h Handler) invitationsClear(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if !httpx.DecodeBody(w, r, &body) {
		return
	}
	n, err := h.Spaces.ClearInvitationCooldowns(r.Context(), httpx.User(r), body.Email)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, clearedBody{Cleared: n})
}
