package authapi

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/routing"
)

func init() {
	routing.Register(nil, func(d *routing.Deps) []routing.Mount {
		h := Handler{Cfg: d.Cfg, Users: d.Users, Sessions: d.Sessions, Tokens: d.PasswordTokens, Challenges: d.Challenges, Settings: d.Settings, Bootstrap: d.Bootstrap, Login: d.Login}
		return []routing.Mount{
			routing.Public(h.PublicRoutes),
			routing.Credential(h.CredentialRoutes),
		}
	})
}

func (h Handler) PublicRoutes(r chi.Router) {
	r.Get("/auth/status", h.authStatus)
	r.Get("/auth/me", h.authMe)
	r.Post("/auth/bootstrap/verify", h.bootstrapVerify)
	r.Post("/auth/register", h.authRegister)
	r.Post("/auth/login", h.authLogin)
	r.Get("/auth/password/{token}", h.passwordPreview)
	r.Post("/auth/password/{token}", h.passwordAccept)
}

func (h Handler) CredentialRoutes(r chi.Router) {
	r.Post("/auth/logout", h.authLogout)
	r.Post("/auth/logout-others", h.authLogoutOthers)
	r.Put("/auth/password", h.authChangePassword)
}
