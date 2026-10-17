package twofactor

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/auth"
	"savvy-go/internal/httpserver/httpx/guard"
	"savvy-go/internal/httpserver/httpx/login"
	"savvy-go/internal/httpserver/httpx/routing"
)

func init() {
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{TwoFactor: routing.Get[auth.TwoFactor](d), Users: routing.Get[auth.Users](d), Challenges: routing.Get[auth.Challenges](d), Login: d.Login, Guard: d.Guard}
		return []routing.Mount{
			routing.Public(h.PublicRoutes),
			routing.Credential(h.CredentialRoutes),
		}
	})
}

// Handler holds what the two-factor routes need.
type Handler struct {
	TwoFactor  auth.TwoFactor
	Users      auth.Users
	Challenges auth.Challenges
	Login      login.Service
	Guard      *guard.Guard
}

func (h Handler) PublicRoutes(r chi.Router) {
	r.With(h.Guard.LimitAttempts).Post("/auth/2fa/verify", h.twoFactorVerify)
}

func (h Handler) CredentialRoutes(r chi.Router) {
	r.Get("/auth/2fa/status", h.twoFactorStatus)
	r.Post("/auth/2fa/enable", h.twoFactorEnable)
	r.Post("/auth/2fa/confirm", h.twoFactorConfirm)
	r.Post("/auth/2fa/disable", h.twoFactorDisable)
	r.Get("/auth/2fa/recovery-codes", h.twoFactorRecoveryCodes)
	r.Post("/auth/2fa/recovery-codes/regenerate", h.twoFactorRegenerate)
}
