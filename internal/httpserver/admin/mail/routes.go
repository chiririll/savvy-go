package mail

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/mail"
)

func init() {
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{Mail: routing.Get[*mail.Service](d), Writable: d.ConfigWritable}
		return []routing.Mount{
			routing.AdminSession(h.Routes),
		}
	})
}

// Handler is the admin's view of the mail settings.
type Handler struct {
	Mail     *mail.Service
	Writable func() bool // whether the config file can be changed
}

func (h Handler) Routes(r chi.Router) {
	r.Get("/admin/mail", h.index)
	r.Put("/admin/mail", h.update)
	r.Post("/admin/mail/test", h.test)
}
