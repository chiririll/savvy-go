package transferapi

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/routing"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	routing.Register(spec, func(d *routing.Deps) []routing.Mount {
		h := Handler{Cfg: d.Cfg, Spaces: d.Spaces, Transfers: d.Transfers, Keys: d.Keys}
		return []routing.Mount{
			routing.AdminSession(h.AdminSessionRoutes),
			routing.SpaceRead(h.ReadRoutes),
			routing.SpaceAdmin(h.AdminRoutes),
			routing.SpaceWrite(h.WriteRoutes),
		}
	})
}

func (h Handler) AdminSessionRoutes(r chi.Router) {
	r.Get("/admin/keys", h.keysIndex)
	r.Post("/admin/keys", h.keysStore)
	r.Delete("/admin/keys/{kid}", h.keysDestroy)
	r.Post("/admin/keys/rotate", h.keysRotate)
}

func (h Handler) ReadRoutes(r chi.Router) {
	r.Get("/links", h.linksIndex)
	r.Get("/transfers", h.transfersIndex)
}

func (h Handler) AdminRoutes(r chi.Router) {
	r.Post("/links", h.linksStore)
	r.Delete("/links/{other}", h.linksDestroy)
}

func (h Handler) WriteRoutes(r chi.Router) {
	// Transfers with linked spaces; the other space's rights
	// are checked by the transfer itself.
	r.Post("/transfers", h.transfersStore)
	r.Patch("/transfers/{uuid}", h.transfersUpdate)
	r.Delete("/transfers/{uuid}", h.transfersDestroy)
	r.Post("/transfers/{uuid}/accept", h.transfersResolve(true))
	r.Post("/transfers/{uuid}/reject", h.transfersResolve(false))
	r.Post("/transfers/{uuid}/trust", h.transfersTrust)
}
