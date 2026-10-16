package automation

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

func Routes(r chi.Router) {
	r.Get("/automation-rules/triggers", automationTriggers)
	r.Post("/automation-rules/reorder", automationReorder)
	r.Get("/automation-rules", automationIndex)
	r.Post("/automation-rules", automationStore)
	r.Get("/automation-rules/{id}", automationShow)
	r.Put("/automation-rules/{id}", automationUpdate)
	r.Patch("/automation-rules/{id}", automationUpdate)
	r.Delete("/automation-rules/{id}", automationDestroy)
	r.Post("/automation-rules/{id}/toggle", automationToggle)
	r.Post("/automation-rules/{id}/test", automationTest)
	r.Get("/automation-rules/{id}/logs", automationLogs)
}
