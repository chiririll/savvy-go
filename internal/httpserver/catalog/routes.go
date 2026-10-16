package catalog

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/routing"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	routing.Register(spec, func(d *routing.Deps) []routing.Mount {
		h := Handler{Rates: d.Rates}
		return []routing.Mount{
			routing.Account(h.Routes),
		}
	})
}

func (h Handler) Routes(r chi.Router) {
	r.Get("/currencies/catalog", h.currencies)
	r.Get("/presets", h.presets)
}
