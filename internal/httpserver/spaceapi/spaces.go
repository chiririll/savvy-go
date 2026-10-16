package spaceapi

import (
	"errors"
	"net/http"

	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/spacefill"
)

func (h Handler) spacesStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		spacefill.Fill
	}
	if !api.DecodeBody(w, r, &body) {
		return
	}
	name, ok := api.SpaceName(w, body.Name)
	if !ok {
		return
	}
	space, err := h.Creator.Create(r.Context(), name, api.User(r), body.Fill)
	var fe *spacefill.Error
	if errors.As(err, &fe) {
		api.WriteValidation(w, map[string][]string{fe.Field: {fe.Msg}})
		return
	}
	if err != nil {
		api.WriteSpaceError(w, err)
		return
	}
	api.WriteData(w, http.StatusCreated, api.SpaceJSON(*space))
}

func (h Handler) spaceShow(w http.ResponseWriter, r *http.Request) {
	space := api.CurrentSpace(w, r, h.Spaces)
	if space == nil {
		return
	}
	space.Role = api.Scope(r).Role
	out := api.SpaceJSON(*space)
	out["quota"] = h.Spaces.Quota(r.Context(), space.ID)
	out["size"], _ = h.Store.SpaceSize(r.Context(), space.ID)
	api.WriteData(w, http.StatusOK, out)
}

func (h Handler) spaceUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if !api.DecodeBody(w, r, &body) {
		return
	}
	name, ok := api.SpaceName(w, body.Name)
	if !ok {
		return
	}
	if err := h.Spaces.Rename(r.Context(), api.Scope(r).ID, name); err != nil {
		api.WriteSpaceError(w, err)
		return
	}
	h.spaceShow(w, r)
}

func (h Handler) spaceDestroy(w http.ResponseWriter, r *http.Request) {
	space := api.CurrentSpace(w, r, h.Spaces)
	if space == nil {
		return
	}
	if err := h.Spaces.Delete(r.Context(), *space, h.Backups); err != nil {
		api.WriteSpaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
