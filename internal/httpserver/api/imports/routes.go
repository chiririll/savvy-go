package imports

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/runtime"
	"savvy-go/internal/uploads"
)

func init() {
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{Uploads: routing.Get[uploads.Service](d), Queue: routing.Get[*runtime.Queue](d)}
		return []routing.Mount{
			routing.SpaceWrite(h.Routes),
		}
	})
}

// Handler serves the import routes. Jobs run on Queue, or at once when it is nil.
type Handler struct {
	Uploads uploads.Service
	Queue   *runtime.Queue
}

func (s Handler) Routes(r chi.Router) {
	r.Post("/transactions/import/parse", s.importParse)
	r.Post("/transactions/import/preview", s.importPreview)
	r.Post("/transactions/import/execute", s.importExecute)
	r.Get("/transactions/import/{import}", s.importShow)
}
