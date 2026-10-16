package sso

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
)

func init() {
	routing.Register(nil, func(d *routing.Deps) []routing.Mount {
		h := Handler{AppURL: d.AppURL, SSO: d.SSO}
		return []routing.Mount{
			routing.Admin(h.Routes),
		}
	})
}

func (h Handler) Routes(r chi.Router) {
	r.Get("/auth/sso/presets", h.ssoPresets)
	r.Get("/identity-providers", h.idpIndex)
	r.Post("/identity-providers", h.idpStore)
	r.Get("/identity-providers/{id}", h.idpShow)
	r.Put("/identity-providers/{id}", h.idpUpdate)
	r.Patch("/identity-providers/{id}", h.idpUpdate)
	r.Delete("/identity-providers/{id}", h.idpDestroy)
	r.Post("/identity-providers/{id}/test", h.idpTest)
}
