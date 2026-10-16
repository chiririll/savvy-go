package budgets

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/routing"
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

// Routes registers the budget routes.
func Routes(r chi.Router) {
	r.Get("/budgets", index)
	r.Post("/budgets", store)
	r.Get("/budgets/{id}", show)
	r.Put("/budgets/{id}", update)
	r.Patch("/budgets/{id}", update)
	r.Delete("/budgets/{id}", destroy)
}
