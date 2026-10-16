// Package space serves spaces: their settings, members and invitations.
package space

import (
	_ "embed"
	"net/http"
	"slices"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/spaces"
)

func (h Handler) index(w http.ResponseWriter, r *http.Request) {
	mine, err := h.Spaces.ForUser(r.Context(), httpx.User(r).ID)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t := httpx.Token(r); t != nil {
		mine = slices.DeleteFunc(mine, func(x spaces.Space) bool { return !t.AllowsSpace(x.ID) })
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(mine, dto.NewSpace))
}
