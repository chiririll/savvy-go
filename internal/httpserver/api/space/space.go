// Package space serves spaces: their settings, members and invitations.
package space

import (
	_ "embed"
	"net/http"
	"slices"

	"savvy-go/internal/auth"
	"savvy-go/internal/backup"
	"savvy-go/internal/config"
	"savvy-go/internal/httpserver/auth/emailverify"
	"savvy-go/internal/httpserver/auth/signin"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/guard"
	"savvy-go/internal/mailing"
	"savvy-go/internal/spacefill"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
)

// Handler holds what the space routes need.
type Handler struct {
	Cfg     config.Config
	Store   store.Store
	Spaces  spaces.Service
	Users   auth.Users
	Backups backup.Service
	Mail    *mailing.Service
	Login   signin.Service
	Verify  emailverify.Handler
	Creator spacefill.Creator
	Guard   *guard.Guard
}

func (h Handler) index(w http.ResponseWriter, r *http.Request) {
	mine, err := h.Spaces.ForUser(r.Context(), httpx.User(r).ID)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t := httpx.Token(r); t != nil {
		mine = slices.DeleteFunc(mine, func(x spaces.Space) bool { return !t.AllowsSpace(x.ID) })
	}
	out := make([]map[string]any, len(mine))
	for i, x := range mine {
		out[i] = httpx.SpaceJSON(x)
	}
	httpx.WriteData(w, http.StatusOK, out)
}
