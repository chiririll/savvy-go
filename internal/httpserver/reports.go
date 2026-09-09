package httpserver

import (
	"net/http"
)

func emptyMetric() map[string]any {
	return map[string]any{"value": 0, "previous": nil, "sparkline": []float64{}}
}

func (s *Server) reportsOverview(w http.ResponseWriter, r *http.Request) {
	sum := s.txs.Summary(r.Context(), false)
	income, _ := sum["income"].(float64)
	expense, _ := sum["expense"].(float64)
	writeJSON(w, http.StatusOK, map[string]any{
		"income":      map[string]any{"value": income, "previous": nil, "sparkline": []float64{}},
		"expenses":    map[string]any{"value": expense, "previous": nil, "sparkline": []float64{}},
		"netCashFlow": map[string]any{"value": income - expense, "previous": nil, "sparkline": []float64{}},
		"savingsRate": emptyMetric(),
		"currency":    sum["currency"],
	})
}

func (s *Server) reportsMoneyFlow(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"nodes": []any{}, "links": []any{},
		"totals": map[string]any{"income": 0, "expenses": 0, "savings": 0},
		"currency": nil,
	})
}

func (s *Server) reportsExpensePace(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"months": []any{}})
}

func (s *Server) reportsByCategory(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"data": []any{}, "total": 0, "currency": nil})
}

func (s *Server) reportsCashFlow(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"periods": []any{}, "currency": nil})
}

func (s *Server) reportsHeatmap(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"days": []any{}, "currency": nil})
}

func (s *Server) reportsTxSummary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.txs.Summary(r.Context(), false))
}

func (s *Server) reportsTxByCategory(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"data": []any{}})
}

func (s *Server) reportsTxDynamics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"data": []any{}})
}

func (s *Server) reportsTxTop(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"data": []any{}})
}

func (s *Server) reportsNetWorth(w http.ResponseWriter, r *http.Request) {
	base, _ := s.currencies.Base(r.Context())
	sum := s.accounts.Summary(r.Context(), base)
	writeJSON(w, http.StatusOK, map[string]any{
		"net_worth": sum["total_balance"], "currency": sum["currency"], "decimals": sum["decimals"],
	})
}

func (s *Server) reportsNetWorthHistory(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"dates": []any{}, "values": []any{}, "currency": nil})
}

func (s *Server) monitoringStorage(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"used_bytes": 0, "total_bytes": nil, "database_bytes": 0})
}

func (s *Server) monitoringResources(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"cpu_percent": nil, "memory_bytes": nil})
}
