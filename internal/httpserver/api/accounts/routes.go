package accounts

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

// Handler serves the account routes; Loc is the server location for default periods.
type Handler struct{ Loc *time.Location }

func (s Handler) Routes(r chi.Router) {
	r.Post("/accounts/reorder", s.accountsReorder)
	r.Get("/accounts", s.accountsIndex)
	r.Post("/accounts", s.accountsStore)
	r.Get("/accounts/{id}", s.accountsShow)
	r.Put("/accounts/{id}", s.accountsUpdate)
	r.Patch("/accounts/{id}", s.accountsUpdate)
	r.Delete("/accounts/{id}", s.accountsDestroy)
	r.Get("/accounts-balance-history", s.accountsBalanceHistory)
	r.Get("/accounts-balance-comparison", s.accountsBalanceComparison)
}
