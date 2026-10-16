package monitoring

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
)

func init() {
	routing.Register(nil, func(d *routing.Deps) []routing.Mount {
		h := Handler{Store: d.Store, Uploads: d.Uploads, UploadsDir: d.Cfg.UploadsDir, Version: d.Build.Version, Env: d.Build.Env}
		return []routing.Mount{
			routing.Admin(h.Routes),
		}
	})
}

func (h Handler) Routes(r chi.Router) {
	r.Get("/monitoring/storage", h.storage)
	r.Get("/monitoring/resources", h.resources)
}
