package emailverify

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/routing"
)

func init() {
	routing.Register(nil, func(d *routing.Deps) []routing.Mount {
		h := FromDeps(d)
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

// FromDeps builds the verification handler from the server's services, for packages that
// also need its CanRequire and Send helpers.
func FromDeps(d *routing.Deps) Handler {
	return Handler{Cfg: d.Cfg, Mail: d.Mail, Verifications: d.Verifications, Users: d.Users}
}
