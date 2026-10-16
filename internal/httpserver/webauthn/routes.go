package webauthn

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/routing"
)

func init() {
	routing.Register(nil, func(d *routing.Deps) []routing.Mount {
		h := Handler{WebAuthn: d.WebAuthn, Challenges: d.Challenges, Login: d.Login, Guard: d.Guard}
		return []routing.Mount{
			routing.Public(h.PublicRoutes),
			routing.Credential(h.CredentialRoutes),
		}
	})
}

func (h Handler) PublicRoutes(r chi.Router) {
	r.With(h.Guard.NeedAppURL).Post("/auth/webauthn/login/options", h.webauthnLoginOptions)
	r.With(h.Guard.NeedAppURL).Post("/auth/webauthn/login/verify", h.webauthnLoginVerify)
}

func (h Handler) CredentialRoutes(r chi.Router) {
	r.Get("/auth/webauthn/credentials", h.webauthnIndex)
	r.With(h.Guard.NeedAppURL).Post("/auth/webauthn/register/options", h.webauthnRegisterOptions)
	r.With(h.Guard.NeedAppURL).Post("/auth/webauthn/register/verify", h.webauthnRegisterVerify)
	r.Patch("/auth/webauthn/credentials/{id}", h.webauthnUpdate)
	r.Delete("/auth/webauthn/credentials/{id}", h.webauthnDestroy)
}
