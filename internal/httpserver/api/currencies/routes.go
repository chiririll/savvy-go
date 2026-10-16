package currencies

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
	r.Get("/currencies/catalog", currenciesCatalog)
	r.Get("/currencies", currenciesIndex)
	r.Post("/currencies", currenciesStore)
	r.Get("/currencies/{id}", currenciesShow)
	r.Put("/currencies/{id}", currenciesUpdate)
	r.Patch("/currencies/{id}", currenciesUpdate)
	r.Delete("/currencies/{id}", currenciesDestroy)
	r.Post("/currencies/{id}/set-base", currenciesSetBase)
	r.Post("/currencies/convert", currenciesConvert)
}
