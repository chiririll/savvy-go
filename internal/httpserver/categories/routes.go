package categories

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/routing"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	routing.Register(spec, func(d *routing.Deps) []routing.Mount {
		h := Handler{Refsync: d.Refsync}
		return []routing.Mount{
			routing.SpaceWrite(h.Routes),
		}
	})
}

func (s Handler) Routes(r chi.Router) {
	r.Get("/categories", s.categoriesIndex)
	r.Post("/categories", s.categoriesStore)
	r.Get("/categories/{id}", s.categoriesShow)
	r.Put("/categories/{id}", s.categoriesUpdate)
	r.Patch("/categories/{id}", s.categoriesUpdate)
	r.Delete("/categories/{id}", s.categoriesDestroy)
	r.Post("/categories/{id}/set-default", s.categoriesSetDefault)
	r.Get("/categories/{id}/statistics", s.categoriesStatistics)
	r.Get("/categories-summary", s.categoriesSummary)
}
