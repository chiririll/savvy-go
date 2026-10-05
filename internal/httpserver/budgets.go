package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"savvy-go/internal/domain"
	"savvy-go/internal/httpserver/dto"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

func (s *Server) budgetsIndex(w http.ResponseWriter, r *http.Request) {
	list, err := sp(r).budgets.All(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, dto.Map(list, dto.NewBudget))
}

func (s *Server) budgetsStore(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeBudget(w, r, nil)
	if !ok {
		return
	}
	if in.Name == "" || !in.Amount.IsPositive() || in.Period == "" {
		writeValidation(w, map[string][]string{"name": {"The name field is required."}})
		return
	}
	b, err := sp(r).budgets.Create(r.Context(), in)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusCreated, dto.NewBudget(*b))
}

func (s *Server) budgetsShow(w http.ResponseWriter, r *http.Request) {
	b := s.budgetParam(w, r)
	if b == nil {
		return
	}
	writeData(w, http.StatusOK, dto.NewBudget(*b))
}

func (s *Server) budgetsUpdate(w http.ResponseWriter, r *http.Request) {
	cur := s.budgetParam(w, r)
	if cur == nil {
		return
	}
	in, ok := decodeBudget(w, r, cur)
	if !ok {
		return
	}
	b, err := sp(r).budgets.Update(r.Context(), cur.ID, in)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusOK, dto.NewBudget(*b))
}

func (s *Server) budgetsDestroy(w http.ResponseWriter, r *http.Request) {
	b := s.budgetParam(w, r)
	if b == nil {
		return
	}
	if err := sp(r).budgets.Delete(r.Context(), b.ID); err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) budgetParam(w http.ResponseWriter, r *http.Request) *domain.Budget {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	b, _ := sp(r).budgets.ByID(r.Context(), id)
	if b == nil {
		writeMessage(w, http.StatusNotFound, "Not found.")
	}
	return b
}

// decodeBudget reads a budget from the body; on update, over base (see
// patch.go).
func decodeBudget(w http.ResponseWriter, r *http.Request, base *domain.Budget) (domain.BudgetInput, bool) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeValidation(w, map[string][]string{"name": {"The name field is required."}})
		return domain.BudgetInput{}, false
	}
	var body struct {
		Name            string          `json:"name"`
		Amount          decimal.Decimal `json:"amount"`
		CurrencyID      *int64          `json:"currency_id"`
		Period          string          `json:"period"`
		StartDate       *string         `json:"start_date"`
		EndDate         *string         `json:"end_date"`
		IsGlobal        *bool           `json:"is_global"`
		NotifyAtPercent *int            `json:"notify_at_percent"`
		IsActive        *bool           `json:"is_active"`
		CategoryIDs     []int64         `json:"category_ids"`
		TagIDs          []int64         `json:"tag_ids"`
	}
	if base != nil {
		body.Name, body.Amount, body.CurrencyID = base.Name, base.Amount.Decimal(), clone(base.CurrencyID)
		body.Period, body.StartDate, body.EndDate = base.Period, clone(base.StartDate), clone(base.EndDate)
		body.IsGlobal, body.NotifyAtPercent = clone(&base.IsGlobal), clone(base.NotifyAtPercent)
		body.IsActive = clone(&base.IsActive)
	}
	buf, _ := json.Marshal(raw)
	if err := json.Unmarshal(buf, &body); err != nil {
		writeValidation(w, map[string][]string{"name": {"The given data was invalid."}})
		return domain.BudgetInput{}, false
	}
	_, hasCats := raw["category_ids"]
	_, hasTags := raw["tag_ids"]
	return domain.BudgetInput{
		Name: body.Name, Amount: body.Amount, CurrencyID: body.CurrencyID, Period: body.Period,
		StartDate: body.StartDate, EndDate: body.EndDate, IsGlobal: body.IsGlobal,
		NotifyAtPercent: body.NotifyAtPercent, IsActive: body.IsActive,
		CategoryIDs: body.CategoryIDs, TagIDs: body.TagIDs,
		HasCategoryIDs: hasCats, HasTagIDs: hasTags,
	}, true
}
