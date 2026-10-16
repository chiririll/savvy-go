package oauth

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/oauth"
)

func init() {
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{OAuth: routing.Get[*oauth.Service](d)}
		return []routing.Mount{
			routing.AdminSession(h.Routes),
		}
	})
}

func (h Handler) Routes(r chi.Router) {
	r.Get("/admin/oauth/registration", h.oauthRegistrationShow)
	r.Post("/admin/oauth/registration", h.oauthRegistrationOpen)
	r.Delete("/admin/oauth/registration", h.oauthRegistrationClose)
	r.Get("/admin/oauth/clients", h.oauthClientsIndex)
	r.Delete("/admin/oauth/clients/{id}", h.oauthClientDestroy)
}
