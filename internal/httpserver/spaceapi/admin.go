package spaceapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/backups"
	"savvy-go/internal/httpserver/dto"
	"savvy-go/internal/spaces"
)

// adminSpacesIndex is the overview of every space: metadata only (P5).
func (h Handler) adminSpacesIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.Spaces.Overview(r.Context())
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	api.WriteData(w, http.StatusOK, dto.Map(list, func(o spaces.Overview) map[string]any {
		return map[string]any{"id": o.ID, "uuid": o.UUID, "name": o.Name, "members": o.Members, "admins": o.Admins,
			"size": o.Size, "quota": o.Quota, "unavailable": o.Unavailable, "createdAt": api.TimeOrNil(o.CreatedAt)}
	}))
}

func (h Handler) adminSpaceParam(w http.ResponseWriter, r *http.Request) *spaces.Space {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err == nil {
		if space, _ := h.Spaces.Get(r.Context(), id); space != nil {
			return space
		}
	}
	api.WriteMessage(w, http.StatusNotFound, "Space not found.")
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
	if !api.DecodeBody(w, r, &body) {
		return
	}
	var bytes int64
	if body.QuotaBytes != nil {
		bytes = *body.QuotaBytes
	}
	if err := h.Spaces.SetQuota(r.Context(), api.User(r), space.ID, bytes); err != nil {
		api.WriteSpaceError(w, err)
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
		api.WriteSpaceError(w, err)
		return
	}
	_ = h.Spaces.Audit(r.Context(), api.User(r), "delete_space", &space.ID, nil, space.Name)
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
	if !api.DecodeBody(w, r, &body) {
		return
	}
	u, _ := h.Users.ByID(r.Context(), body.UserID)
	if u == nil || u.Deleted {
		api.WriteMessage(w, http.StatusNotFound, "User not found.")
		return
	}
	if err := h.Spaces.AssignAdmin(r.Context(), api.User(r), space.ID, u); err != nil {
		api.WriteSpaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) adminDeletedIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.Backups.DeletedBackups()
	backups.List(w, list, err)
}

// adminDeletedRestore brings a deleted space back as a new space owned by the
// calling server admin.
func (h Handler) adminDeletedRestore(w http.ResponseWriter, r *http.Request) {
	path, err := h.Backups.DeletedPath(chi.URLParam(r, "name"))
	if err != nil {
		backups.WriteError(w, err)
		return
	}
	space, err := h.Backups.ImportSpace(r.Context(), h.Spaces, path, "", api.User(r), 0)
	if err != nil {
		backups.WriteError(w, err)
		return
	}
	_ = h.Spaces.Audit(r.Context(), api.User(r), "restore_deleted_space", &space.ID, nil, space.Name)
	api.WriteData(w, http.StatusCreated, api.SpaceJSON(*space))
}
