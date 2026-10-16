package space

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/auth/emailverify"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/spacefill"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	routing.Register(spec, func(d *routing.Deps) []routing.Mount {
		h := Handler{
			Cfg: d.Cfg, Store: d.Store, Spaces: d.Spaces, Users: d.Users, Backups: d.Backups, Mail: d.Mail,
			Login: d.Login, Verify: emailverify.FromDeps(d), Guard: d.Guard,
			Creator: spacefill.Creator{Spaces: d.Spaces, Store: d.Store, Rates: d.Rates},
		}
		return []routing.Mount{
			routing.Public(h.PublicRoutes),
			routing.Account(h.AccountRoutes),
			routing.SpaceRead(h.ReadRoutes),
			routing.SpaceAdmin(h.AdminRoutes),
		}
	})
}

func (h Handler) PublicRoutes(r chi.Router) {
	r.Get("/invitations/{token}", h.invitationPreview)
	r.Post("/invitations/{token}/register", h.invitationRegister)
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

func (h Handler) AccountRoutes(r chi.Router) {
	r.Get("/spaces", h.index)
	r.Get("/invitation-methods", h.invitationMethods)
	r.With(h.Guard.SessionOnly).Post("/spaces", h.spacesStore)
	r.With(h.Guard.SessionOnly).Post("/invitations/{token}/accept", h.invitationAccept)
}
