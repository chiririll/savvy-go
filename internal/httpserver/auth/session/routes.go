package session

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/auth"
	"savvy-go/internal/firstrun"
	"savvy-go/internal/httpserver/httpx/guard"
	"savvy-go/internal/httpserver/httpx/login"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/settings"
)

func init() {
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{
			Security:   routing.Get[auth.Security](d),
			Users:      routing.Get[auth.Users](d),
			Sessions:   routing.Get[auth.Sessions](d),
			APITokens:  routing.Get[auth.APITokens](d),
			Tokens:     routing.Get[auth.PasswordTokens](d),
			Challenges: routing.Get[auth.Challenges](d),
			Settings:   routing.Get[settings.Store](d),
			Bootstrap:  routing.Get[*firstrun.Guard](d),
			Login:      d.Login,
			Guard:      d.Guard,
		}
		return []routing.Mount{
			routing.Public(h.PublicRoutes),
			routing.Credential(h.CredentialRoutes),
		}
	})
}

// Handler holds what the account routes need.
type Handler struct {
	Security   auth.Security
	Users      auth.Users
	Sessions   auth.Sessions
	APITokens  auth.APITokens
	Tokens     auth.PasswordTokens
	Challenges auth.Challenges
	Settings   settings.Store
	Bootstrap  *firstrun.Guard
	Login      login.Service
	Guard      *guard.Guard
}

func (h Handler) PublicRoutes(r chi.Router) {
	r.Get("/auth/status", h.authStatus)
	r.Get("/auth/me", h.authMe)

	// These take guesses at a password or a token, so they are throttled.
	limited := r.With(h.Guard.LimitAttempts)
	limited.Post("/auth/bootstrap/verify", h.bootstrapVerify)
	limited.Post("/auth/register", h.authRegister)
	limited.Post("/auth/login", h.authLogin)
	limited.Get("/auth/password/{token}", h.passwordPreview)
	limited.Post("/auth/password/{token}", h.passwordAccept)
}

func (h Handler) CredentialRoutes(r chi.Router) {
	r.Post("/auth/logout", h.authLogout)
	r.Post("/auth/logout-others", h.authLogoutOthers)
	r.Put("/auth/password", h.authChangePassword)
}
