package recurring

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	routing.Register(spec, func(d *routing.Deps) []routing.Mount {
		return []routing.Mount{
			routing.SpaceWrite(Routes),
		}
	})
}

func Routes(r chi.Router) {
	r.Get("/recurring", recurringIndex)
	r.Get("/recurring-upcoming", recurringUpcoming)
	r.Post("/recurring", recurringStore)
	r.Get("/recurring/{id}", recurringShow)
	r.Put("/recurring/{id}", recurringUpdate)
	r.Patch("/recurring/{id}", recurringUpdate)
	r.Delete("/recurring/{id}", recurringDestroy)
}
