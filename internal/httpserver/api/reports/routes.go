package reports

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
	r.Get("/reports/overview", reportsOverview)
	r.Get("/reports/money-flow", reportsMoneyFlow)
	r.Get("/reports/expense-pace", reportsExpensePace)
	r.Get("/reports/expenses-by-category", reportsByCategory)
	r.Get("/reports/cash-flow-over-time", reportsCashFlow)
	r.Get("/reports/activity-heatmap", reportsHeatmap)
	r.Get("/reports/transactions/summary", reportsTxSummary)
	r.Get("/reports/transactions/by-category", reportsTxByCategory)
	r.Get("/reports/transactions/dynamics", reportsTxDynamics)
	r.Get("/reports/transactions/top", reportsTxTop)
	r.Get("/reports/net-worth", reportsNetWorth)
	r.Get("/reports/net-worth-history", reportsNetWorthHistory)
}
