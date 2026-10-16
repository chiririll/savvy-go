package mail

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
)

func init() {
	routing.Register(nil, func(d *routing.Deps) []routing.Mount {
		h := Handler{Mail: d.Mail, Writable: d.ConfigWritable}
		return []routing.Mount{
			routing.AdminSession(h.Routes),
		}
	})
}

func (h Handler) Routes(r chi.Router) {
	r.Get("/admin/mail", h.index)
	r.Post("/admin/mail/test", h.test)
}
