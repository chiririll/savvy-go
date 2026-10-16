package accounts

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/routing"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	routing.Register(spec, func(d *routing.Deps) []routing.Mount {
		h := Handler{Loc: d.Cfg.Location}
		return []routing.Mount{
			routing.SpaceWrite(h.Routes),
		}
	})
}

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
