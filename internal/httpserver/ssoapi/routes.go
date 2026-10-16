package ssoapi

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/routing"
)

func init() {
	routing.Register(nil, func(d *routing.Deps) []routing.Mount {
		h := Handler{Cfg: d.Cfg, SSO: d.SSO, Users: d.Users, Challenges: d.Challenges, Login: d.Login, Guard: d.Guard}
		return []routing.Mount{
			routing.Public(h.PublicRoutes),
			routing.Admin(h.AdminRoutes),
		}
	})
}

func (h Handler) PublicRoutes(r chi.Router) {
	r.Get("/auth/sso/providers", h.ssoProviders)
	r.Post("/auth/sso/exchange", h.ssoExchange)
	r.With(h.Guard.NeedAppURL).Get("/auth/sso/{slug}/redirect", h.ssoRedirect)
	r.With(h.Guard.NeedAppURL).Get("/auth/sso/{slug}/callback", h.ssoCallback)
	r.With(h.Guard.NeedAppURL).Post("/auth/sso/{slug}/acs", h.ssoACS)
	r.With(h.Guard.NeedAppURL).Get("/auth/sso/{slug}/metadata", h.ssoMetadata)
}

func (h Handler) AdminRoutes(r chi.Router) {
	r.Get("/auth/sso/presets", h.ssoPresets)
	r.Get("/identity-providers", h.idpIndex)
	r.Post("/identity-providers", h.idpStore)
	r.Get("/identity-providers/{id}", h.idpShow)
	r.Put("/identity-providers/{id}", h.idpUpdate)
	r.Patch("/identity-providers/{id}", h.idpUpdate)
	r.Delete("/identity-providers/{id}", h.idpDestroy)
	r.Post("/identity-providers/{id}/test", h.idpTest)
}
