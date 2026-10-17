// Package reports serves /reports of a space.
package reports

import (
	_ "embed"
	"net/http"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
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
	httpx.WriteJSON(w, http.StatusOK, dto.NewOverview(httpx.Scope(r).Reports.Overview(r.Context(), reportFilter(r))))
}

func reportsMoneyFlow(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, dto.NewMoneyFlow(httpx.Scope(r).Reports.MoneyFlow(r.Context(), reportFilter(r))))
}

func reportsExpensePace(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, dto.NewExpensePace(httpx.Scope(r).Reports.ExpensePace(r.Context(), reportFilter(r))))
}

func reportsByCategory(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, dto.NewExpensesByCategory(httpx.Scope(r).Reports.ExpensesByCategory(r.Context(), reportFilter(r))))
}

func reportsCashFlow(w http.ResponseWriter, r *http.Request) {
	group := firstQuery(r.URL.Query().Get("group_by"), "day")
	httpx.WriteJSON(w, http.StatusOK, dto.NewCashFlow(httpx.Scope(r).Reports.CashFlowOverTime(r.Context(), reportFilter(r), group)))
}

func reportsHeatmap(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, dto.NewHeatmap(httpx.Scope(r).Reports.Heatmap(r.Context(), reportFilter(r))))
}

func reportsTxSummary(w http.ResponseWriter, r *http.Request) {
	typ := firstQuery(r.URL.Query().Get("type"), "expense")
	httpx.WriteJSON(w, http.StatusOK, dto.NewTxSummary(httpx.Scope(r).Reports.TxSummary(r.Context(), reportFilter(r), typ)))
}

func reportsTxByCategory(w http.ResponseWriter, r *http.Request) {
	typ := firstQuery(r.URL.Query().Get("type"), "expense")
	httpx.WriteJSON(w, http.StatusOK, dto.NewTxByCategory(httpx.Scope(r).Reports.TxByCategory(r.Context(), reportFilter(r), typ)))
}

func reportsTxDynamics(w http.ResponseWriter, r *http.Request) {
	typ := firstQuery(r.URL.Query().Get("type"), "expense")
	group := firstQuery(r.URL.Query().Get("group_by"), "day")
	httpx.WriteJSON(w, http.StatusOK, dto.NewTxDynamics(httpx.Scope(r).Reports.TxDynamics(r.Context(), reportFilter(r), typ, group)))
}

func reportsTxTop(w http.ResponseWriter, r *http.Request) {
	typ := firstQuery(r.URL.Query().Get("type"), "expense")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	httpx.WriteJSON(w, http.StatusOK, dto.NewTxTop(httpx.Scope(r).Reports.TxTop(r.Context(), reportFilter(r), typ, limit)))
}

func reportsNetWorth(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, dto.NewNetWorth(httpx.Scope(r).Reports.NetWorth(r.Context(), reportFilter(r))))
}

func reportsNetWorthHistory(w http.ResponseWriter, r *http.Request) {
	group := firstQuery(r.URL.Query().Get("group_by"), "day")
	httpx.WriteJSON(w, http.StatusOK, dto.NewNetWorthHistory(httpx.Scope(r).Reports.NetWorthHistory(r.Context(), reportFilter(r), group)))
}
