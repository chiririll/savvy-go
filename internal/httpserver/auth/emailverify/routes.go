package emailverify

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
)

func init() {
	routing.Register(nil, func(d *routing.Deps) []routing.Mount {
		h := Handler{Verify: d.EmailVerify}
		return []routing.Mount{
			routing.Public(h.PublicRoutes),
			routing.Credential(h.CredentialRoutes),
		}
	})
}

func (h Handler) PublicRoutes(r chi.Router) {
	r.Post("/auth/verify-email", h.verifyEmail)
}

func (h Handler) CredentialRoutes(r chi.Router) {
	r.Post("/auth/verify-email/resend", h.verifyEmailResend)
}
