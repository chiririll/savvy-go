package transactions

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/httpserver/ui"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	ui.AddSpec(spec)
	routing.Register(func(d *routing.Deps) []routing.Mount {
		return []routing.Mount{
			routing.SpaceWrite(Routes),
		}
	})
}

func Routes(r chi.Router) {
	r.Get("/transactions", transactionsIndex)
	r.Post("/transactions", transactionsStore)
	r.Post("/transactions/reorder", transactionsReorder)
	r.Get("/transactions/{id}", transactionsShow)
	r.Put("/transactions/{id}", transactionsUpdate)
	r.Patch("/transactions/{id}", transactionsUpdate)
	r.Delete("/transactions/{id}", transactionsDestroy)
	r.Post("/transactions/{id}/duplicate", transactionsDuplicate)
	r.Post("/transactions/{id}/confirm", transactionsConfirm)
	r.Post("/transactions/{id}/skip", transactionsSkip)
	r.Post("/transaction-groups", groupsStore)
	r.Get("/transaction-groups/{uuid}", groupsShow)
	r.Put("/transaction-groups/{uuid}", groupsUpdate)
	r.Delete("/transaction-groups/{uuid}", groupsDestroy)
	r.Get("/transactions-summary", transactionsSummary)
	r.Get("/transactions-pending-summary", transactionsPendingSummary)
}
