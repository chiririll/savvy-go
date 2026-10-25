// Package budgets serves /budgets of a space.
package budgets

import (
	_ "embed"
	"net/http"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/ledger"

	"github.com/shopspring/decimal"
)

func index(w http.ResponseWriter, r *http.Request) {
	list, err := httpx.Scope(r).Budgets.All(r.Context())
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(list, dto.NewBudget))
}

func store(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeBudget(w, r, nil)
	if !ok {
		return
	}
	if in.Name == "" || !in.Amount.IsPositive() || in.Period == "" {
		httpx.WriteValidation(w, map[string][]string{"name": {"The name field is required."}})
		return
	}
	b, err := httpx.Scope(r).Budgets.Create(r.Context(), in)
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusCreated, dto.NewBudget(*b))
}

func show(w http.ResponseWriter, r *http.Request) {
	b := param(w, r)
	if b == nil {
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewBudget(*b))
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
	b, err := httpx.Scope(r).Budgets.Update(r.Context(), cur.ID, in)
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewBudget(*b))
}

func destroy(w http.ResponseWriter, r *http.Request) {
	b := param(w, r)
	if b == nil {
		return
	}
	if err := httpx.Scope(r).Budgets.Delete(r.Context(), b.ID); err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func param(w http.ResponseWriter, r *http.Request) *ledger.Budget {
	return httpx.Lookup(w, r, httpx.Scope(r).Budgets.ByID)
}

// decodeBudget reads a budget from the body; on update, over base: fields the
// body leaves out keep their values, and category_ids / tag_ids are replaced
// only when sent.
func decodeBudget(w http.ResponseWriter, r *http.Request, base *ledger.Budget) (ledger.BudgetInput, bool) {
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
		body.Name, body.Amount, body.CurrencyID = base.Name, base.Amount.Decimal(), httpx.Clone(base.CurrencyID)
		body.Period, body.StartDate, body.EndDate = base.Period, httpx.Clone(base.StartDate), httpx.Clone(base.EndDate)
		body.IsGlobal, body.NotifyAtPercent = httpx.Clone(&base.IsGlobal), httpx.Clone(base.NotifyAtPercent)
		body.IsActive = httpx.Clone(&base.IsActive)
	}
	sent, ok := httpx.DecodeSent(w, r, &body, "name")
	if !ok {
		return ledger.BudgetInput{}, false
	}
	_, hasCats := sent["category_ids"]
	_, hasTags := sent["tag_ids"]
	return ledger.BudgetInput{
		Name: body.Name, Amount: body.Amount, CurrencyID: body.CurrencyID, Period: body.Period,
		StartDate: body.StartDate, EndDate: body.EndDate, IsGlobal: body.IsGlobal,
		NotifyAtPercent: body.NotifyAtPercent, IsActive: body.IsActive,
		CategoryIDs: body.CategoryIDs, TagIDs: body.TagIDs,
		HasCategoryIDs: hasCats, HasTagIDs: hasTags,
	}, true
}
