package keys

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
)

func init() {
	routing.Register(nil, func(d *routing.Deps) []routing.Mount {
		h := Handler{Cfg: d.Cfg, Transfers: d.Transfers, Keys: d.Keys}
		return []routing.Mount{
			routing.AdminSession(h.Routes),
		}
	})
}

func (h Handler) Routes(r chi.Router) {
	r.Get("/admin/keys", h.keysIndex)
	r.Post("/admin/keys", h.keysStore)
	r.Delete("/admin/keys/{kid}", h.keysDestroy)
	r.Post("/admin/keys/rotate", h.keysRotate)
}
