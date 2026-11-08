package reconciliations

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/httpserver/ui"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	ui.AddSpec(spec)
	routing.Register(func(d *routing.Deps) []routing.Mount {
		return []routing.Mount{
			routing.SpaceWrite(Routes),
		}
	})
}

func Routes(r chi.Router) {
	r.Get("/reconciliations", index)
	r.Post("/reconciliations", store)
	r.Delete("/reconciliations/{id}", destroy)
	r.Get("/reconciliations-days", days)
	r.Get("/accounts/{id}/reconciliations", accountIndex)
}
