package categories

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/httpserver/ui"
	"savvy-go/internal/refsync"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	ui.AddSpec(spec)
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{Refsync: routing.Get[refsync.Service](d)}
		return []routing.Mount{
			routing.SpaceWrite(h.Routes),
		}
	})
}

// Handler serves the categories routes; changes go through the sync groups.
type Handler struct{ Refsync refsync.Service }

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
