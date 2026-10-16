// Package budgets serves /budgets of a space.
package budgets

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"strconv"

	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/apidocs"
	"savvy-go/internal/httpserver/dto"
	"savvy-go/internal/ledger"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

func index(w http.ResponseWriter, r *http.Request) {
	list, err := api.Scope(r).Budgets.All(r.Context())
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	api.WriteData(w, http.StatusOK, dto.Map(list, dto.NewBudget))
}

func store(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeBudget(w, r, nil)
	if !ok {
		return
	}
	if in.Name == "" || !in.Amount.IsPositive() || in.Period == "" {
		api.WriteValidation(w, map[string][]string{"name": {"The name field is required."}})
		return
	}
	b, err := api.Scope(r).Budgets.Create(r.Context(), in)
	if err != nil {
		api.WriteMessage(w, 422, err.Error())
		return
	}
	api.WriteData(w, http.StatusCreated, dto.NewBudget(*b))
}

func show(w http.ResponseWriter, r *http.Request) {
	b := param(w, r)
	if b == nil {
		return
	}
	api.WriteData(w, http.StatusOK, dto.NewBudget(*b))
}

func update(w http.ResponseWriter, r *http.Request) {
	cur := param(w, r)
	if cur == nil {
		return
	}
	in, ok := decodeBudget(w, r, cur)
	if !ok {
		return
	}
	b, err := api.Scope(r).Budgets.Update(r.Context(), cur.ID, in)
	if err != nil {
		api.WriteMessage(w, 422, err.Error())
		return
	}
	api.WriteData(w, http.StatusOK, dto.NewBudget(*b))
}

func destroy(w http.ResponseWriter, r *http.Request) {
	b := param(w, r)
	if b == nil {
		return
	}
	if err := api.Scope(r).Budgets.Delete(r.Context(), b.ID); err != nil {
		api.WriteMessage(w, 422, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func param(w http.ResponseWriter, r *http.Request) *ledger.Budget {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	b, _ := api.Scope(r).Budgets.ByID(r.Context(), id)
	if b == nil {
		api.WriteMessage(w, http.StatusNotFound, "Not found.")
	}
	return b
}

// decodeBudget reads a budget from the body; on update, over base (see
// patch.go).
func decodeBudget(w http.ResponseWriter, r *http.Request, base *ledger.Budget) (ledger.BudgetInput, bool) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		api.WriteValidation(w, map[string][]string{"name": {"The name field is required."}})
		return ledger.BudgetInput{}, false
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
		body.Name, body.Amount, body.CurrencyID = base.Name, base.Amount.Decimal(), api.Clone(base.CurrencyID)
		body.Period, body.StartDate, body.EndDate = base.Period, api.Clone(base.StartDate), api.Clone(base.EndDate)
		body.IsGlobal, body.NotifyAtPercent = api.Clone(&base.IsGlobal), api.Clone(base.NotifyAtPercent)
		body.IsActive = api.Clone(&base.IsActive)
	}
	buf, _ := json.Marshal(raw)
	if err := json.Unmarshal(buf, &body); err != nil {
		api.WriteValidation(w, map[string][]string{"name": {"The given data was invalid."}})
		return ledger.BudgetInput{}, false
	}
	_, hasCats := raw["category_ids"]
	_, hasTags := raw["tag_ids"]
	return ledger.BudgetInput{
		Name: body.Name, Amount: body.Amount, CurrencyID: body.CurrencyID, Period: body.Period,
		StartDate: body.StartDate, EndDate: body.EndDate, IsGlobal: body.IsGlobal,
		NotifyAtPercent: body.NotifyAtPercent, IsActive: body.IsActive,
		CategoryIDs: body.CategoryIDs, TagIDs: body.TagIDs,
		HasCategoryIDs: hasCats, HasTagIDs: hasTags,
	}, true
}

//go:embed openapi.yaml
var spec []byte

// Routes registers the budget routes.
func Routes(r chi.Router) {
	apidocs.Add(spec)
	r.Get("/budgets", index)
	r.Post("/budgets", store)
	r.Get("/budgets/{id}", show)
	r.Put("/budgets/{id}", update)
	r.Patch("/budgets/{id}", update)
	r.Delete("/budgets/{id}", destroy)
}
