package twofactor

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
)

func init() {
	routing.Register(nil, func(d *routing.Deps) []routing.Mount {
		h := Handler{TwoFactor: d.TwoFactor, Users: d.Users, Challenges: d.Challenges, Login: d.Login}
		return []routing.Mount{
			routing.Public(h.PublicRoutes),
			routing.Credential(h.CredentialRoutes),
		}
	})
}

func (h Handler) PublicRoutes(r chi.Router) {
	r.Post("/auth/2fa/verify", h.twoFactorVerify)
}

func (h Handler) CredentialRoutes(r chi.Router) {
	r.Get("/auth/2fa/status", h.twoFactorStatus)
	r.Post("/auth/2fa/enable", h.twoFactorEnable)
	r.Post("/auth/2fa/confirm", h.twoFactorConfirm)
	r.Post("/auth/2fa/disable", h.twoFactorDisable)
	r.Get("/auth/2fa/recovery-codes", h.twoFactorRecoveryCodes)
	r.Post("/auth/2fa/recovery-codes/regenerate", h.twoFactorRegenerate)
}
