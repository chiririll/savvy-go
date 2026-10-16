package transfer

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/httpserver/ui"
	"savvy-go/internal/signing"
	"savvy-go/internal/spaces"
	"savvy-go/internal/transfers"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	ui.AddSpec(spec)
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{Spaces: routing.Get[spaces.Service](d), Transfers: routing.Get[*transfers.Service](d), Keys: routing.Get[*signing.Holder](d)}
		return []routing.Mount{
			routing.SpaceRead(h.ReadRoutes),
			routing.SpaceAdmin(h.AdminRoutes),
			routing.SpaceWrite(h.WriteRoutes),
		}
	})
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
