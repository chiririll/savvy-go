package debts

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
	r.Get("/debts", debtsIndex)
	r.Post("/debts", debtsStore)
	r.Get("/debts/{id}", debtsShow)
	r.Put("/debts/{id}", debtsUpdate)
	r.Patch("/debts/{id}", debtsUpdate)
	r.Delete("/debts/{id}", debtsDestroy)
	r.Post("/debts/{id}/payment", debtsPayment)
	r.Post("/debts/{id}/collect", debtsCollect)
	r.Post("/debts/{id}/reopen", debtsReopen)
	r.Get("/debts-summary", debtsSummary)
}
