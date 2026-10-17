package health

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/store"
)

func init() {
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{Store: routing.Get[store.Store](d), Version: routing.Get[appconfig.Build](d).Version}
		return []routing.Mount{
			routing.Root(h.Routes),
		}
	})
}

// Handler serves the liveness and readiness probes.
type Handler struct {
	Store   store.Store
	Version string // the release, reported as releaseId
}

// Routes mounts the probes outside /api.
func (h Handler) Routes(r chi.Router) {
	r.Get("/livez", h.Live)
	r.Get("/readyz", h.Ready)
}
