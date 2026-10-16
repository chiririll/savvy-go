// Package accounts serves /accounts of a space.
package accounts

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/dto"
	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
	"savvy-go/internal/reports"
)

// Handler serves the account routes; Loc is the server location for default periods.
type Handler struct{ Loc *time.Location }

func (s Handler) accountsIndex(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("active") == "1" || r.URL.Query().Get("active") == "true"
	excludeDebts := r.URL.Query().Get("exclude_debts") == "1" || r.URL.Query().Get("exclude_debts") == "true"
	list, err := api.Scope(r).Accounts.All(r.Context(), onlyActive, excludeDebts)
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	payload := api.Envelope{Data: dto.Map(list, dto.NewAccount)}
	if r.URL.Query().Get("with_summary") == "1" || r.URL.Query().Get("with_summary") == "true" {
		base, _ := api.Scope(r).Currencies.Base(r.Context())
		if id := r.URL.Query().Get("base_currency_id"); id != "" {
			if n, err := strconv.ParseInt(id, 10, 64); err == nil {
				base, _ = api.Scope(r).Currencies.ByID(r.Context(), n)
			}
		}
		payload.Meta = nil
		api.WriteJSON(w, http.StatusOK, map[string]any{
			"data":    payload.Data,
			"summary": dto.NewAccountsSummary(api.Scope(r).Accounts.Summary(r.Context(), base)),
		})
		return
	}
	api.WriteData(w, http.StatusOK, payload.Data)
}

func (s Handler) accountsStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name           string          `json:"name"`
		Type           string          `json:"type"`
		CurrencyID     *int64          `json:"currency_id"`
		CurrencyCode   string          `json:"currency_code"`
		InitialBalance decimal.Decimal `json:"initial_balance"`
		IsActive       *bool           `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		api.WriteValidation(w, map[string][]string{"name": {"The name field is required."}})
		return
	}
	var currencyID int64
	if body.CurrencyID != nil {
		currencyID = *body.CurrencyID
	} else if body.CurrencyCode != "" {
		c, err := api.Scope(r).Currencies.FindOrCreateByCode(r.Context(), body.CurrencyCode)
		if err != nil || c == nil {
			api.WriteValidation(w, map[string][]string{"currency_code": {"Unknown currency code."}})
			return
		}
		currencyID = c.ID
	} else {
		api.WriteValidation(w, map[string][]string{"currency_id": {"The currency id field is required."}})
		return
	}
	active := true
	if body.IsActive != nil {
		active = *body.IsActive
	}
	a, err := api.Scope(r).Accounts.Create(r.Context(), ledger.AccountInput{
		Name: body.Name, Type: body.Type, CurrencyID: currencyID,
		InitialBalance: body.InitialBalance, IsActive: active,
	})
	if err != nil {
		api.WriteAccountError(w, err)
		return
	}
	api.WriteData(w, http.StatusCreated, dto.NewAccount(*a))
}

func (s Handler) accountsShow(w http.ResponseWriter, r *http.Request) {
	a := s.accountParam(w, r)
	if a == nil {
		return
	}
	api.WriteData(w, http.StatusOK, dto.NewAccount(*a))
}

func (s Handler) accountsUpdate(w http.ResponseWriter, r *http.Request) {
	cur := s.accountParam(w, r)
	if cur == nil {
		return
	}
	var body struct {
		Name           *string          `json:"name"`
		Type           *string          `json:"type"`
		CurrencyID     *int64           `json:"currency_id"`
		CurrencyCode   *string          `json:"currency_code"`
		InitialBalance *decimal.Decimal `json:"initial_balance"`
		IsActive       *bool            `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteValidation(w, map[string][]string{"name": {"The given data was invalid."}})
		return
	}
	in := cur.Input()
	if body.Name != nil {
		in.Name = *body.Name
	}
	if body.Type != nil {
		in.Type = *body.Type
	}
	if body.CurrencyID != nil {
		in.CurrencyID = *body.CurrencyID
	}
	// A code is only resolved, never created: the currency cannot change anyway.
	if body.CurrencyCode != nil && *body.CurrencyCode != "" {
		c, err := api.Scope(r).Currencies.ByCode(r.Context(), *body.CurrencyCode)
		if err != nil || c == nil || c.ID != in.CurrencyID {
			api.WriteAccountError(w, ledger.ErrAccountCurrencyImmutable)
			return
		}
	}
	if body.InitialBalance != nil {
		in.InitialBalance = *body.InitialBalance
	}
	if body.IsActive != nil {
		in.IsActive = *body.IsActive
	}
	a, err := api.Scope(r).Accounts.Update(r.Context(), cur.ID, in)
	if err != nil {
		api.WriteAccountError(w, err)
		return
	}
	api.WriteData(w, http.StatusOK, dto.NewAccount(*a))
}

func (s Handler) accountsReorder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteValidation(w, map[string][]string{"ids": {"The ids field is required."}})
		return
	}
	if err := api.Scope(r).Accounts.Reorder(r.Context(), body.IDs); err != nil {
		api.WriteMessage(w, 422, err.Error())
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s Handler) accountsDestroy(w http.ResponseWriter, r *http.Request) {
	a := s.accountParam(w, r)
	if a == nil {
		return
	}
	if err := api.Scope(r).Accounts.Delete(r.Context(), a.ID); err != nil {
		api.WriteMessage(w, 422, "Cannot delete account that has transactions.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Handler) accountsBalanceHistory(w http.ResponseWriter, r *http.Request) {
	base, _ := api.Scope(r).Currencies.Base(r.Context())
	start, end := s.queryPeriod(r)
	if base == nil {
		api.WriteJSON(w, http.StatusOK, map[string]any{"dates": []string{}, "series": []any{}, "currency": nil, "decimals": 2})
		return
	}
	accts, _ := api.Scope(r).Accounts.All(r.Context(), true, true)
	dates := dateRange(start, end)
	series := []map[string]any{}
	total := make([]money.Money, len(dates))
	for i := range total {
		total[i] = money.Zero(base.Unit())
	}
	for _, a := range accts {
		native, err := api.Scope(r).Accounts.BalanceSeries(r.Context(), a, dates) // in the account's own currency
		if err != nil {
			api.WriteMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
		data := make([]money.Money, len(dates)) // in the base currency
		for i, bal := range native {
			data[i] = ledger.Convert(bal, *a.Currency, *base)
			total[i] = total[i].Add(data[i])
		}
		series = append(series, map[string]any{
			"id": a.ID, "name": a.Name, "type": a.Type, "data": data, "native_data": native, "currency": a.Currency.Code,
		})
	}
	if len(accts) > 1 {
		series = append([]map[string]any{{"id": nil, "name": "Total", "type": "total", "data": total, "currency": base.Code}}, series...)
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"dates": dates, "series": series, "currency": base.Code, "decimals": base.Decimals,
	})
}

// queryPeriod resolves start_date/end_date from the query, falling back to the
// shared default report period when either is missing or invalid.
func (s Handler) queryPeriod(r *http.Request) (start, end string) {
	q := r.URL.Query()
	rng := reports.Filter{
		PeriodType: "custom",
		StartDate:  q.Get("start_date"),
		EndDate:    q.Get("end_date"),
	}.Range(time.Now().In(s.Loc))
	return rng.Start.Format("2006-01-02"), rng.End.Format("2006-01-02")
}

// accountsBalanceComparison returns the current total balance and the total
// balance at the end of the day before the period starts.
func (s Handler) accountsBalanceComparison(w http.ResponseWriter, r *http.Request) {
	base, _ := api.Scope(r).Currencies.Base(r.Context())
	sum := dto.NewAccountsSummary(api.Scope(r).Accounts.Summary(r.Context(), base))
	var previous *money.Money
	if base != nil {
		start, _ := s.queryPeriod(r)
		if from, err := time.Parse("2006-01-02", start); err == nil {
			total := api.Scope(r).Accounts.TotalAt(r.Context(), *base, from.AddDate(0, 0, -1).Format("2006-01-02"))
			previous = &total
		}
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"current":  sum.TotalBalance,
		"previous": previous,
		"currency": sum.Currency,
		"decimals": sum.Decimals,
	})
}

func (s Handler) accountParam(w http.ResponseWriter, r *http.Request) *ledger.Account {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	a, _ := api.Scope(r).Accounts.ByID(r.Context(), id)
	if a == nil {
		api.WriteMessage(w, http.StatusNotFound, "Not found.")
	}
	return a
}

func dateRange(start, end string) []string {
	from, err1 := time.Parse("2006-01-02", start)
	to, err2 := time.Parse("2006-01-02", end)
	if err1 != nil || err2 != nil {
		return nil
	}
	var out []string
	for !from.After(to) {
		out = append(out, from.Format("2006-01-02"))
		from = from.AddDate(0, 0, 1)
	}
	return out
}

func (s Handler) Routes(r chi.Router) {
	r.Post("/accounts/reorder", s.accountsReorder)
	r.Get("/accounts", s.accountsIndex)
	r.Post("/accounts", s.accountsStore)
	r.Get("/accounts/{id}", s.accountsShow)
	r.Put("/accounts/{id}", s.accountsUpdate)
	r.Patch("/accounts/{id}", s.accountsUpdate)
	r.Delete("/accounts/{id}", s.accountsDestroy)
	r.Get("/accounts-balance-history", s.accountsBalanceHistory)
	r.Get("/accounts-balance-comparison", s.accountsBalanceComparison)
}
