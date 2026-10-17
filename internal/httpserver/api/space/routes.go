package space

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/auth"
	"savvy-go/internal/backup"
	"savvy-go/internal/httpserver/httpx/guard"
	"savvy-go/internal/httpserver/httpx/login"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/httpserver/ui"
	"savvy-go/internal/ledger"
	"savvy-go/internal/mail"
	"savvy-go/internal/spaces"
	"savvy-go/internal/spaces/spacefill"
	"savvy-go/internal/store"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	ui.AddSpec(spec)
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{
			Cfg: routing.Get[appconfig.Config](
				d,
			), Store: routing.Get[store.Store](d), Spaces: routing.Get[spaces.Service](d), Users: routing.Get[auth.Users](d), Backups: routing.Get[backup.Service](d), Mail: routing.Get[*mail.Service](d),
			Login: d.Login, Verify: routing.Get[mail.Verifier](d), Guard: d.Guard,
			Creator: spacefill.Creator{
				Spaces: routing.Get[spaces.Service](d),
				Store:  routing.Get[store.Store](d),
				Rates:  routing.Get[ledger.RateSource](d),
			},
		}
		return []routing.Mount{
			routing.Public(h.PublicRoutes),
			routing.Account(h.AccountRoutes),
			routing.SpaceRead(h.ReadRoutes),
			routing.SpaceAdmin(h.AdminRoutes),
		}
	})
}

// Handler holds what the space routes need.
type Handler struct {
	Cfg     appconfig.Config
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

func (h Handler) PublicRoutes(r chi.Router) {
	limited := r.With(h.Guard.LimitAttempts)
	limited.Get("/invitations/{token}", h.invitationPreview)
	limited.Post("/invitations/{token}/register", h.invitationRegister)
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
