package webauthn

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/auth"
	"savvy-go/internal/httpserver/httpx/guard"
	"savvy-go/internal/httpserver/httpx/login"
	"savvy-go/internal/httpserver/httpx/routing"
)

func init() {
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{WebAuthn: routing.Get[auth.WebAuthn](d), Challenges: routing.Get[auth.Challenges](d), Login: d.Login, Guard: d.Guard}
		return []routing.Mount{
			routing.Public(h.PublicRoutes),
			routing.Credential(h.CredentialRoutes),
		}
	})
}

// Handler holds what the passkey routes need.
type Handler struct {
	WebAuthn   auth.WebAuthn
	Challenges auth.Challenges
	Login      login.Service
	Guard      *guard.Guard
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
