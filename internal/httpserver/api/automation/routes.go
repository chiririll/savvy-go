package automation

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
	r.Get("/automation-rules/triggers", triggers)
	r.Post("/automation-rules/reorder", reorder)
	r.Get("/automation-rules", index)
	r.Post("/automation-rules", store)
	r.Get("/automation-rules/{id}", show)
	r.Put("/automation-rules/{id}", update)
	r.Patch("/automation-rules/{id}", update)
	r.Delete("/automation-rules/{id}", destroy)
	r.Post("/automation-rules/{id}/toggle", toggle)
	r.Post("/automation-rules/{id}/test", test)
	r.Get("/automation-rules/{id}/logs", logs)
}
