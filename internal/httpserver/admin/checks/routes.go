package checks

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/backup"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/mail"
)

func init() {
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{Cfg: routing.Get[appconfig.Config](d), Writable: d.ConfigWritable, UploadsDir: routing.Get[appconfig.Config](d).UploadsDir, Backups: routing.Get[backup.Service](d), Mail: routing.Get[*mail.Service](d).Status}
		return []routing.Mount{
			routing.Admin(h.Routes),
		}
	})
}

func (h Handler) Routes(r chi.Router) {
	r.Get("/admin/checks", h.index)
}
