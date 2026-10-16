package imports

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/routing"
)

func init() {
	routing.Register(nil, func(d *routing.Deps) []routing.Mount {
		h := Handler{Uploads: d.Uploads, Queue: d.Queue}
		return []routing.Mount{
			routing.SpaceWrite(h.Routes),
		}
	})
}

func (s Handler) Routes(r chi.Router) {
	r.Post("/transactions/import/parse", s.importParse)
	r.Post("/transactions/import/preview", s.importPreview)
	r.Post("/transactions/import/execute", s.importExecute)
	r.Get("/transactions/import/{import}", s.importShow)
}
