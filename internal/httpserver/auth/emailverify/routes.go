package emailverify

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/guard"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/mail"
)

func init() {
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{Verify: routing.Get[mail.Verifier](d), Guard: d.Guard}
		return []routing.Mount{
			routing.Public(h.PublicRoutes),
			routing.Credential(h.CredentialRoutes),
		}
	})
}

// Handler holds what the confirmation routes need.
type Handler struct {
	Verify mail.Verifier
	Guard  *guard.Guard
}

func (h Handler) PublicRoutes(r chi.Router) {
	r.With(h.Guard.LimitAttempts).Post("/auth/verify-email", h.verifyEmail)
}

func (h Handler) CredentialRoutes(r chi.Router) {
	r.Post("/auth/verify-email/resend", h.verifyEmailResend)
}
