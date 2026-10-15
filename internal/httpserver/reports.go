package httpserver

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
	"strings"

	"savvy-go/internal/reports"
)

func (s *Server) reportFilter(r *http.Request) reports.Filter {
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

func (s *Server) reportsOverview(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, sp(r).reports.Overview(r.Context(), s.reportFilter(r)))
}

func (s *Server) reportsMoneyFlow(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, sp(r).reports.MoneyFlow(r.Context(), s.reportFilter(r)))
}

func (s *Server) reportsExpensePace(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, sp(r).reports.ExpensePace(r.Context(), s.reportFilter(r)))
}

func (s *Server) reportsByCategory(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, sp(r).reports.ExpensesByCategory(r.Context(), s.reportFilter(r)))
}

func (s *Server) reportsCashFlow(w http.ResponseWriter, r *http.Request) {
	group := firstQuery(r.URL.Query().Get("group_by"), "day")
	writeJSON(w, http.StatusOK, sp(r).reports.CashFlowOverTime(r.Context(), s.reportFilter(r), group))
}

func (s *Server) reportsHeatmap(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, sp(r).reports.Heatmap(r.Context(), s.reportFilter(r)))
}

func (s *Server) reportsTxSummary(w http.ResponseWriter, r *http.Request) {
	typ := firstQuery(r.URL.Query().Get("type"), "expense")
	writeJSON(w, http.StatusOK, sp(r).reports.TxSummary(r.Context(), s.reportFilter(r), typ))
}

func (s *Server) reportsTxByCategory(w http.ResponseWriter, r *http.Request) {
	typ := firstQuery(r.URL.Query().Get("type"), "expense")
	writeJSON(w, http.StatusOK, sp(r).reports.TxByCategory(r.Context(), s.reportFilter(r), typ))
}

func (s *Server) reportsTxDynamics(w http.ResponseWriter, r *http.Request) {
	typ := firstQuery(r.URL.Query().Get("type"), "expense")
	group := firstQuery(r.URL.Query().Get("group_by"), "day")
	writeJSON(w, http.StatusOK, sp(r).reports.TxDynamics(r.Context(), s.reportFilter(r), typ, group))
}

func (s *Server) reportsTxTop(w http.ResponseWriter, r *http.Request) {
	typ := firstQuery(r.URL.Query().Get("type"), "expense")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	writeJSON(w, http.StatusOK, sp(r).reports.TxTop(r.Context(), s.reportFilter(r), typ, limit))
}

func (s *Server) reportsNetWorth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, sp(r).reports.NetWorth(r.Context(), s.reportFilter(r)))
}

func (s *Server) reportsNetWorthHistory(w http.ResponseWriter, r *http.Request) {
	group := firstQuery(r.URL.Query().Get("group_by"), "day")
	writeJSON(w, http.StatusOK, sp(r).reports.NetWorthHistory(r.Context(), s.reportFilter(r), group))
}

func (s *Server) reportRoutes(r chi.Router) {
	r.Get("/reports/overview", s.reportsOverview)
	r.Get("/reports/money-flow", s.reportsMoneyFlow)
	r.Get("/reports/expense-pace", s.reportsExpensePace)
	r.Get("/reports/expenses-by-category", s.reportsByCategory)
	r.Get("/reports/cash-flow-over-time", s.reportsCashFlow)
	r.Get("/reports/activity-heatmap", s.reportsHeatmap)
	r.Get("/reports/transactions/summary", s.reportsTxSummary)
	r.Get("/reports/transactions/by-category", s.reportsTxByCategory)
	r.Get("/reports/transactions/dynamics", s.reportsTxDynamics)
	r.Get("/reports/transactions/top", s.reportsTxTop)
	r.Get("/reports/net-worth", s.reportsNetWorth)
	r.Get("/reports/net-worth-history", s.reportsNetWorthHistory)
}
