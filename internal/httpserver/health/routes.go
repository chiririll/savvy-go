package health

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/routing"
)

func init() {
	routing.Register(nil, func(d *routing.Deps) []routing.Mount {
		h := Handler{Store: d.Store}
		return []routing.Mount{
			routing.Root(h.Routes),
		}
	})
}

// Routes mounts the probes outside /api.
func (h Handler) Routes(r chi.Router) {
	r.Get("/livez", h.Live)
	r.Get("/readyz", h.Ready)
}
