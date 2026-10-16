package space

import (
	"errors"
	"net/http"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/spacefill"
)

func (h Handler) spacesStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		spacefill.Fill
	}
	if !httpx.DecodeBody(w, r, &body) {
		return
	}
	name, ok := httpx.SpaceName(w, body.Name)
	if !ok {
		return
	}
	space, err := h.Creator.Create(r.Context(), name, httpx.User(r), body.Fill)
	var fe *spacefill.Error
	if errors.As(err, &fe) {
		httpx.WriteValidation(w, map[string][]string{fe.Field: {fe.Msg}})
		return
	}
	if err != nil {
		httpx.WriteSpaceError(w, err)
		return
	}
	httpx.WriteData(w, http.StatusCreated, dto.NewSpace(*space))
}

func (h Handler) spaceShow(w http.ResponseWriter, r *http.Request) {
	space := httpx.CurrentSpace(w, r, h.Spaces)
	if space == nil {
		return
	}
	space.Role = httpx.Scope(r).Role
	size, _ := h.Store.SpaceSize(r.Context(), space.ID)
	httpx.WriteData(w, http.StatusOK, dto.SpaceDetail{
		Space: dto.NewSpace(*space), Quota: h.Spaces.Quota(r.Context(), space.ID), Size: size,
	})
}

func (h Handler) spaceUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if !httpx.DecodeBody(w, r, &body) {
		return
	}
	name, ok := httpx.SpaceName(w, body.Name)
	if !ok {
		return
	}
	if err := h.Spaces.Rename(r.Context(), httpx.Scope(r).ID, name); err != nil {
		httpx.WriteSpaceError(w, err)
		return
	}
	h.spaceShow(w, r)
}

func (h Handler) spaceDestroy(w http.ResponseWriter, r *http.Request) {
	space := httpx.CurrentSpace(w, r, h.Spaces)
	if space == nil {
		return
	}
	if err := h.Spaces.Delete(r.Context(), *space, h.Backups); err != nil {
		httpx.WriteSpaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
