package transactions

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	routing.Register(spec, func(d *routing.Deps) []routing.Mount {
		return []routing.Mount{
			routing.SpaceWrite(Routes),
		}
	})
}

func Routes(r chi.Router) {
	r.Get("/transactions", transactionsIndex)
	r.Post("/transactions", transactionsStore)
	r.Get("/transactions/{id}", transactionsShow)
	r.Put("/transactions/{id}", transactionsUpdate)
	r.Patch("/transactions/{id}", transactionsUpdate)
	r.Delete("/transactions/{id}", transactionsDestroy)
	r.Post("/transactions/{id}/duplicate", transactionsDuplicate)
	r.Post("/transactions/{id}/confirm", transactionsConfirm)
	r.Post("/transactions/{id}/skip", transactionsSkip)
	r.Get("/transactions-summary", transactionsSummary)
	r.Get("/transactions-pending-summary", transactionsPendingSummary)
}
