// Package reports serves /reports of a space.
package reports

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"savvy-go/internal/httpserver/api"
	"strconv"
	"strings"

	"savvy-go/internal/reports"
)

func reportFilter(r *http.Request) reports.Filter {
	q := r.URL.Query()
	return reports.Filter{
		PeriodType:  firstQuery(q.Get("period_type"), "last_30_days"),
		PeriodValue: q.Get("period_value"),
		StartDate:   q.Get("start_date"),
		EndDate:     q.Get("end_date"),
		CompareWith: firstQuery(q.Get("compare_with"), "none"),
		AccountIDs:  queryIDs(q, "account_ids"),
		CategoryIDs: queryIDs(q, "category_ids"),
		TagIDs:      queryIDs(q, "tag_ids"),
	}
}

func firstQuery(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func queryIDs(q map[string][]string, key string) []int64 {
	var raw []string
	raw = append(raw, q[key]...)
	raw = append(raw, q[key+"[]"]...)
	var out []int64
	for _, v := range raw {
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			n, err := strconv.ParseInt(part, 10, 64)
			if err == nil {
				out = append(out, n)
			}
		}
	}
	return out
}

func reportsOverview(w http.ResponseWriter, r *http.Request) {
	api.WriteJSON(w, http.StatusOK, api.Scope(r).Reports.Overview(r.Context(), reportFilter(r)))
}

func reportsMoneyFlow(w http.ResponseWriter, r *http.Request) {
	api.WriteJSON(w, http.StatusOK, api.Scope(r).Reports.MoneyFlow(r.Context(), reportFilter(r)))
}

func reportsExpensePace(w http.ResponseWriter, r *http.Request) {
	api.WriteJSON(w, http.StatusOK, api.Scope(r).Reports.ExpensePace(r.Context(), reportFilter(r)))
}

func reportsByCategory(w http.ResponseWriter, r *http.Request) {
	api.WriteJSON(w, http.StatusOK, api.Scope(r).Reports.ExpensesByCategory(r.Context(), reportFilter(r)))
}

func reportsCashFlow(w http.ResponseWriter, r *http.Request) {
	group := firstQuery(r.URL.Query().Get("group_by"), "day")
	api.WriteJSON(w, http.StatusOK, api.Scope(r).Reports.CashFlowOverTime(r.Context(), reportFilter(r), group))
}

func reportsHeatmap(w http.ResponseWriter, r *http.Request) {
	api.WriteJSON(w, http.StatusOK, api.Scope(r).Reports.Heatmap(r.Context(), reportFilter(r)))
}

func reportsTxSummary(w http.ResponseWriter, r *http.Request) {
	typ := firstQuery(r.URL.Query().Get("type"), "expense")
	api.WriteJSON(w, http.StatusOK, api.Scope(r).Reports.TxSummary(r.Context(), reportFilter(r), typ))
}

func reportsTxByCategory(w http.ResponseWriter, r *http.Request) {
	typ := firstQuery(r.URL.Query().Get("type"), "expense")
	api.WriteJSON(w, http.StatusOK, api.Scope(r).Reports.TxByCategory(r.Context(), reportFilter(r), typ))
}

func reportsTxDynamics(w http.ResponseWriter, r *http.Request) {
	typ := firstQuery(r.URL.Query().Get("type"), "expense")
	group := firstQuery(r.URL.Query().Get("group_by"), "day")
	api.WriteJSON(w, http.StatusOK, api.Scope(r).Reports.TxDynamics(r.Context(), reportFilter(r), typ, group))
}

func reportsTxTop(w http.ResponseWriter, r *http.Request) {
	typ := firstQuery(r.URL.Query().Get("type"), "expense")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	api.WriteJSON(w, http.StatusOK, api.Scope(r).Reports.TxTop(r.Context(), reportFilter(r), typ, limit))
}

func reportsNetWorth(w http.ResponseWriter, r *http.Request) {
	api.WriteJSON(w, http.StatusOK, api.Scope(r).Reports.NetWorth(r.Context(), reportFilter(r)))
}

func reportsNetWorthHistory(w http.ResponseWriter, r *http.Request) {
	group := firstQuery(r.URL.Query().Get("group_by"), "day")
	api.WriteJSON(w, http.StatusOK, api.Scope(r).Reports.NetWorthHistory(r.Context(), reportFilter(r), group))
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
