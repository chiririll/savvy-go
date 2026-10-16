package checks

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
)

func init() {
	routing.Register(nil, func(d *routing.Deps) []routing.Mount {
		h := Handler{Cfg: d.Cfg, Writable: d.ConfigWritable, UploadsDir: d.Cfg.UploadsDir, Backups: d.Backups, Mail: d.Mail.Status}
		return []routing.Mount{
			routing.Admin(h.Routes),
		}
	})
}

func (h Handler) Routes(r chi.Router) {
	r.Get("/admin/checks", h.index)
}
