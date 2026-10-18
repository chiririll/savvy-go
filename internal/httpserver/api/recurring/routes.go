package recurring

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
	r.Get("/recurring", index)
	r.Get("/recurring-upcoming", upcoming)
	r.Post("/recurring", store)
	r.Get("/recurring/{id}", show)
	r.Put("/recurring/{id}", update)
	r.Patch("/recurring/{id}", update)
	r.Delete("/recurring/{id}", destroy)
}
