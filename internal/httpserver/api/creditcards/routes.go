package creditcards

import (
	_ "embed"
	"time"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/httpserver/ui"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	ui.AddSpec(spec)
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{Loc: routing.Get[appconfig.Config](d).Location}
		return []routing.Mount{
			routing.SpaceWrite(h.Routes),
		}
	})
}

// Handler serves the credit card routes; Loc is the server location, which
// decides what today is for a statement.
type Handler struct{ Loc *time.Location }

func (s Handler) Routes(r chi.Router) {
	r.Get("/credit-cards", s.index)
	r.Post("/credit-cards", s.store)
	r.Get("/credit-cards/{id}", s.show)
	r.Put("/credit-cards/{id}", s.update)
	r.Patch("/credit-cards/{id}", s.update)
	r.Delete("/credit-cards/{id}", s.destroy)
}
