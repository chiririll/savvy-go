// Package space serves spaces: their settings, members and invitations.
package space

import (
	_ "embed"
	"net/http"
	"slices"

	"savvy-go/internal/auth"
	"savvy-go/internal/backup"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/httpserver/httpx/guard"
	"savvy-go/internal/httpserver/httpx/login"
	"savvy-go/internal/mail"
	"savvy-go/internal/spacefill"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
)

// Handler holds what the space routes need.
type Handler struct {
	AppURL  func() string // the public URL of the instance, empty when unset
	Store   store.Store
	Spaces  spaces.Service
	Users   auth.Users
	Backups backup.Service
	Mail    *mail.Service
	Login   login.Service
	Verify  mail.Verifier
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
	httpx.WriteData(w, http.StatusOK, dto.Map(mine, dto.NewSpace))
}
