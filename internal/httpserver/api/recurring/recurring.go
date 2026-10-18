// Package recurring serves /recurring of a space.
package recurring

import (
	_ "embed"
	"net/http"
	"savvy-go/internal/httpserver/httpx"

	"github.com/shopspring/decimal"

	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/recurring"
)

func index(w http.ResponseWriter, r *http.Request) {
	list, err := httpx.Scope(r).Recurring.All(r.Context())
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(list, dto.NewRecurring))
}

func upcoming(w http.ResponseWriter, r *http.Request) {
	list, err := httpx.Scope(r).Recurring.Upcoming(r.Context(), 5)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(list, dto.NewRecurring))
}

func store(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeRecurring(w, r, nil)
	if !ok {
		return
	}
	rec, err := httpx.Scope(r).Recurring.Create(r.Context(), in)
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusCreated, dto.NewRecurring(*rec))
}

func show(w http.ResponseWriter, r *http.Request) {
	rec := param(w, r)
	if rec == nil {
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewRecurring(*rec))
}

func update(w http.ResponseWriter, r *http.Request) {
	cur := param(w, r)
	if cur == nil {
		return
	}
	in, ok := decodeRecurring(w, r, cur)
	if !ok {
		return
	}
	rec, err := httpx.Scope(r).Recurring.Update(r.Context(), cur.ID, in)
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewRecurring(*rec))
}

func destroy(w http.ResponseWriter, r *http.Request) {
	rec := param(w, r)
	if rec == nil {
		return
	}
	if err := httpx.Scope(r).Recurring.Delete(r.Context(), rec.ID); err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func param(w http.ResponseWriter, r *http.Request) *recurring.Recurring {
	return httpx.Lookup(w, r, httpx.Scope(r).Recurring.ByID)
}

// decodeRecurring reads a recurring template from the body. On update, base
// is the stored template: fields the body leaves out keep its values, and an
// explicit null clears a nullable one.
func decodeRecurring(w http.ResponseWriter, r *http.Request, base *recurring.Recurring) (recurring.Input, bool) {
	var body struct {
		Type        string           `json:"type"`
		AccountID   int64            `json:"account_id"`
		ToAccountID *int64           `json:"to_account_id"`
		CategoryID  *int64           `json:"category_id"`
		Amount      decimal.Decimal  `json:"amount"`
		ToAmount    *decimal.Decimal `json:"to_amount"`
		IsEstimated *bool            `json:"is_estimated"`
		Description *string          `json:"description"`
		Frequency   string           `json:"frequency"`
		Interval    int              `json:"interval"`
		DayOfWeek   *int             `json:"day_of_week"`
		DayOfMonth  *int             `json:"day_of_month"`
		StartDate   string           `json:"start_date"`
		EndDate     *string          `json:"end_date"`
		IsActive    *bool            `json:"is_active"`
		TagIDs      []int64          `json:"tag_ids"`
	}
	if base != nil {
		body.Type, body.AccountID, body.ToAccountID = base.Type, base.AccountID, httpx.Clone(base.ToAccountID)
		body.CategoryID, body.Amount = httpx.Clone(base.CategoryID), base.Amount.Decimal()
		if base.ToAmount != nil {
			toAmount := base.ToAmount.Decimal()
			body.ToAmount = &toAmount
		}
		body.IsEstimated, body.Description = httpx.Clone(&base.IsEstimated), httpx.Clone(base.Description)
		body.Frequency, body.Interval = base.Frequency, base.Interval
		body.DayOfWeek, body.DayOfMonth = httpx.Clone(base.DayOfWeek), httpx.Clone(base.DayOfMonth)
		body.StartDate, body.EndDate, body.IsActive = base.StartDate, httpx.Clone(base.EndDate), httpx.Clone(&base.IsActive)
	}
	sent, ok := httpx.DecodeSent(w, r, &body, "type")
	if !ok {
		return recurring.Input{}, false
	}
	_, hasTags := sent["tag_ids"]
	return recurring.Input{
		Type: body.Type, AccountID: body.AccountID, ToAccountID: body.ToAccountID,
		CategoryID: body.CategoryID, Amount: body.Amount, ToAmount: body.ToAmount,
		IsEstimated: body.IsEstimated, Description: body.Description, Frequency: body.Frequency, Interval: body.Interval,
		DayOfWeek: body.DayOfWeek, DayOfMonth: body.DayOfMonth, StartDate: body.StartDate,
		EndDate: body.EndDate, IsActive: body.IsActive, TagIDs: body.TagIDs, HasTagIDs: hasTags,
	}, true
}
