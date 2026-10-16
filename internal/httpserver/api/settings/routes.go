package settings

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	routing.Register(spec, func(d *routing.Deps) []routing.Mount {
		h := Handler{Cfg: d.Cfg, Settings: d.Settings, SSO: d.SSO, Verify: d.EmailVerify}
		return []routing.Mount{
			routing.AdminSession(h.AdminSessionRoutes),
			routing.Account(h.AccountRoutes),
			routing.SpaceRead(h.SpaceReadRoutes),
			routing.SpaceAdmin(h.SpaceAdminRoutes),
		}
	})
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
