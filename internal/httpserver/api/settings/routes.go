package settings

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/httpserver/ui"
	"savvy-go/internal/mail"
	"savvy-go/internal/settings"
	"savvy-go/internal/sso"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	ui.AddSpec(spec)
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{Cfg: routing.Get[appconfig.Config](d), Writable: d.ConfigWritable, Settings: routing.Get[settings.Store](d), SSO: routing.Get[*sso.Service](d), Verify: routing.Get[mail.Verifier](d)}
		return []routing.Mount{
			routing.AdminSession(h.AdminSessionRoutes),
			routing.Account(h.AccountRoutes),
			routing.SpaceRead(h.SpaceReadRoutes),
			routing.SpaceAdmin(h.SpaceAdminRoutes),
		}
	})
}

// Handler holds what the settings routes need.
type Handler struct {
	Cfg      appconfig.Config
	Writable func() bool // whether the config file can be changed
	Settings settings.Store
	SSO      *sso.Service
	Verify   mail.Verifier
}

func (h Handler) AdminSessionRoutes(r chi.Router) {
	r.Patch("/settings", h.settingsUpdate)
}

func (h Handler) SpaceReadRoutes(r chi.Router) {
	r.Get("/settings", h.spaceSettingsIndex)
}

func (h Handler) SpaceAdminRoutes(r chi.Router) {
	r.Patch("/settings", h.spaceSettingsUpdate)
}

func (h Handler) AccountRoutes(r chi.Router) {
	// Instance settings everyone may read (sign-in options).
	r.Get("/settings", h.settingsIndex)
}
