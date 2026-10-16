package users

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	routing.Register(spec, func(d *routing.Deps) []routing.Mount {
		h := Handler{AppURL: d.AppURL, Users: d.Users, Tokens: d.PasswordTokens, Spaces: d.Spaces, Mail: d.Mail, Backups: d.Backups, Login: d.Login}
		return []routing.Mount{
			routing.Admin(h.Routes),
		}
	})
}

func (h Handler) Routes(r chi.Router) {
	r.Get("/users", h.usersIndex)
	r.Get("/users/{id}", h.usersShow)
	r.Post("/users", h.usersStore)
	r.Post("/users/{id}/password-token", h.usersIssueToken)
	r.Put("/users/{id}", h.usersUpdate)
	r.Patch("/users/{id}", h.usersUpdate)
	r.Delete("/users/{id}", h.usersDestroy)
}
