package sso

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/auth"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/sso"
)

func init() {
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{Cfg: routing.Get[appconfig.Config](d), SSO: routing.Get[*sso.Service](d), Users: routing.Get[auth.Users](d), Challenges: routing.Get[auth.Challenges](d), Login: d.Login, Guard: d.Guard}
		return []routing.Mount{
			routing.Public(h.PublicRoutes),
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
