package httpserver

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/chiririll/savvy-plus/internal/domain"
)

func (s *Server) reportFilter(r *http.Request) domain.ReportFilter {
	q := r.URL.Query()
	return domain.ReportFilter{
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
	writeJSON(w, http.StatusOK, s.reports.Overview(r.Context(), s.reportFilter(r)))
}

func (s *Server) reportsMoneyFlow(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.reports.MoneyFlow(r.Context(), s.reportFilter(r)))
}

func (s *Server) reportsExpensePace(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.reports.ExpensePace(r.Context(), s.reportFilter(r)))
}

func (s *Server) reportsByCategory(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.reports.ExpensesByCategory(r.Context(), s.reportFilter(r)))
}

func (s *Server) reportsCashFlow(w http.ResponseWriter, r *http.Request) {
	group := firstQuery(r.URL.Query().Get("group_by"), "day")
	writeJSON(w, http.StatusOK, s.reports.CashFlowOverTime(r.Context(), s.reportFilter(r), group))
}

func (s *Server) reportsHeatmap(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.reports.Heatmap(r.Context(), s.reportFilter(r)))
}

func (s *Server) reportsTxSummary(w http.ResponseWriter, r *http.Request) {
	typ := firstQuery(r.URL.Query().Get("type"), "expense")
	writeJSON(w, http.StatusOK, s.reports.TxSummary(r.Context(), s.reportFilter(r), typ))
}

func (s *Server) reportsTxByCategory(w http.ResponseWriter, r *http.Request) {
	typ := firstQuery(r.URL.Query().Get("type"), "expense")
	writeJSON(w, http.StatusOK, s.reports.TxByCategory(r.Context(), s.reportFilter(r), typ))
}

func (s *Server) reportsTxDynamics(w http.ResponseWriter, r *http.Request) {
	typ := firstQuery(r.URL.Query().Get("type"), "expense")
	group := firstQuery(r.URL.Query().Get("group_by"), "day")
	writeJSON(w, http.StatusOK, s.reports.TxDynamics(r.Context(), s.reportFilter(r), typ, group))
}

func (s *Server) reportsTxTop(w http.ResponseWriter, r *http.Request) {
	typ := firstQuery(r.URL.Query().Get("type"), "expense")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	writeJSON(w, http.StatusOK, s.reports.TxTop(r.Context(), s.reportFilter(r), typ, limit))
}

func (s *Server) reportsNetWorth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.reports.NetWorth(r.Context(), s.reportFilter(r)))
}

func (s *Server) reportsNetWorthHistory(w http.ResponseWriter, r *http.Request) {
	group := firstQuery(r.URL.Query().Get("group_by"), "day")
	writeJSON(w, http.StatusOK, s.reports.NetWorthHistory(r.Context(), s.reportFilter(r), group))
}
