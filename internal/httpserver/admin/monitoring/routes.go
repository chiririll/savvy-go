package monitoring

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/store"
	"savvy-go/internal/uploads"
)

func init() {
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{
			Store:      routing.Get[store.Store](d),
			Uploads:    routing.Get[uploads.Service](d),
			UploadsDir: routing.Get[appconfig.Config](d).UploadsDir,
			Version:    routing.Get[appconfig.Build](d).Version,
			Env:        routing.Get[appconfig.Build](d).Env,
		}
		return []routing.Mount{
			routing.Admin(h.Routes),
		}
	})
}

// Handler is the monitoring of the instance: disk, uploads, imports, process.
type Handler struct {
	Store      store.Store
	Uploads    uploads.Service
	UploadsDir string
	Version    string // the release
	Env        string // the build environment
}

func (h Handler) Routes(r chi.Router) {
	r.Get("/monitoring/storage", h.storage)
	r.Get("/monitoring/resources", h.resources)
}
