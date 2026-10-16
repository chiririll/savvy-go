package tags

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

func (s Handler) Routes(r chi.Router) {
	r.Get("/tags", s.tagsIndex)
	r.Post("/tags", s.tagsStore)
	r.Get("/tags/{id}", s.tagsShow)
	r.Put("/tags/{id}", s.tagsUpdate)
	r.Patch("/tags/{id}", s.tagsUpdate)
	r.Delete("/tags/{id}", s.tagsDestroy)
}
