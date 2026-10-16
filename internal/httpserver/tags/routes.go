package tags

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
	r.Get("/tags", s.tagsIndex)
	r.Post("/tags", s.tagsStore)
	r.Get("/tags/{id}", s.tagsShow)
	r.Put("/tags/{id}", s.tagsUpdate)
	r.Patch("/tags/{id}", s.tagsUpdate)
	r.Delete("/tags/{id}", s.tagsDestroy)
}
