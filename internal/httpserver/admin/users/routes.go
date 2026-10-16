package users

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/auth"
	"savvy-go/internal/backup"
	"savvy-go/internal/httpserver/httpx/login"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/httpserver/ui"
	"savvy-go/internal/mail"
	"savvy-go/internal/spaces"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	ui.AddSpec(spec)
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{Cfg: routing.Get[appconfig.Config](d), Users: routing.Get[auth.Users](d), Tokens: routing.Get[auth.PasswordTokens](d), Spaces: routing.Get[spaces.Service](d), Mail: routing.Get[*mail.Service](d), Backups: routing.Get[backup.Service](d), Login: d.Login}
		return []routing.Mount{
			routing.Admin(h.Routes),
		}
	})
}

// Handler holds what the user administration needs.
type Handler struct {
	Cfg     appconfig.Config
	Users   auth.Users
	Tokens  auth.PasswordTokens
	Spaces  spaces.Service
	Mail    *mail.Service
	Backups backup.Service
	Login   login.Service
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
