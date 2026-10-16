// Package spaceapi serves spaces: their settings, members, invitations and the server admin's view of them.
package spaceapi

import (
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/auth"
	"savvy-go/internal/backup"
	"savvy-go/internal/config"
	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/emailverify"
	"savvy-go/internal/httpserver/guard"
	"savvy-go/internal/httpserver/mailing"
	"savvy-go/internal/httpserver/signin"
	"savvy-go/internal/httpserver/spacefill"
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

func (h Handler) PublicRoutes(r chi.Router) {
	r.Get("/invitations/{token}", h.invitationPreview)
	r.Post("/invitations/{token}/register", h.invitationRegister)
}

func (h Handler) AdminSessionRoutes(r chi.Router) {
	r.Post("/admin/invitations/clear", h.invitationsClear)
	r.Get("/admin/spaces", h.adminSpacesIndex)
	r.Patch("/admin/spaces/{id}", h.adminSpaceUpdate)
	r.Delete("/admin/spaces/{id}", h.adminSpaceDestroy)
	r.Post("/admin/spaces/{id}/admins", h.adminSpaceAssignAdmin)
	r.Get("/admin/deleted-spaces", h.adminDeletedIndex)
	r.Post("/admin/deleted-spaces/{name}/restore", h.adminDeletedRestore)
}

func (h Handler) ReadRoutes(r chi.Router) {
	r.Get("/", h.spaceShow)
	r.Get("/members", h.membersIndex)
	r.With(h.Guard.SessionOnly).Post("/leave", h.spaceLeave)
}

func (h Handler) AdminRoutes(r chi.Router) {
	r.Patch("/", h.spaceUpdate)
	r.Delete("/", h.spaceDestroy)
	r.Patch("/members/{user}", h.membersUpdate)
	r.Delete("/members/{user}", h.membersDestroy)
	r.Get("/invitations", h.invitationsIndex)
	r.Post("/invitations", h.invitationsStore)
	r.Delete("/invitations/{id}", h.invitationsDestroy)
	r.Get("/audit", h.spaceAudit)
}

func (h Handler) index(w http.ResponseWriter, r *http.Request) {
	mine, err := h.Spaces.ForUser(r.Context(), api.User(r).ID)
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t := api.Token(r); t != nil {
		mine = slices.DeleteFunc(mine, func(x spaces.Space) bool { return !t.AllowsSpace(x.ID) })
	}
	out := make([]map[string]any, len(mine))
	for i, x := range mine {
		out[i] = api.SpaceJSON(x)
	}
	api.WriteData(w, http.StatusOK, out)
}

func (h Handler) AccountRoutes(r chi.Router) {
	r.Get("/spaces", h.index)
	r.Get("/invitation-methods", h.invitationMethods)
	r.With(h.Guard.SessionOnly).Post("/spaces", h.spacesStore)
	r.With(h.Guard.SessionOnly).Post("/invitations/{token}/accept", h.invitationAccept)
}
