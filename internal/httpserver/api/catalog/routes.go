package catalog

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/httpserver/ui"
	"savvy-go/internal/ledger"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	ui.AddSpec(spec)
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{Rates: routing.Get[ledger.RateSource](d)}
		return []routing.Mount{
			routing.Account(h.Routes),
		}
	})
}

// Handler serves what a new space can start with, before the space exists.
type Handler struct{ Rates ledger.RateSource }

func (h Handler) Routes(r chi.Router) {
	// Under /catalog, not /currencies: the SPA prefixes space roots with /spaces/{id}.
	r.Get("/catalog/currencies", h.currencies)
	r.Get("/catalog/presets", h.presets)
}
