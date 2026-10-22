package keys

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/signing"
	"savvy-go/internal/transfers"
)

func init() {
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{
			DataDir:   routing.Get[appconfig.Config](d).DataDir,
			Transfers: routing.Get[*transfers.Service](d),
			Keys:      routing.Get[*signing.Holder](d),
		}
		return []routing.Mount{
			routing.AdminSession(h.Routes),
		}
	})
}

// Handler holds what the key routes need.
type Handler struct {
	DataDir   string
	Transfers *transfers.Service
	Keys      *signing.Holder
}

func (h Handler) Routes(r chi.Router) {
	r.Get("/admin/keys", h.keysIndex)
	r.Post("/admin/keys", h.keysStore)
	r.Delete("/admin/keys/{kid}", h.keysDestroy)
	r.Post("/admin/keys/rotate", h.keysRotate)
}
