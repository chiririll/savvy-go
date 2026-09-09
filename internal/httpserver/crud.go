package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chiririll/savvy-plus/internal/domain"
	"github.com/go-chi/chi/v5"
)

func (s *Server) currenciesIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.currencies.All(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, mapSlice(list, domain.Currency.JSON))
}

func (s *Server) currenciesCatalog(w http.ResponseWriter, r *http.Request) {
	writeData(w, http.StatusOK, s.currencies.Catalog(r.Context()))
}

func (s *Server) currenciesStore(w http.ResponseWriter, r *http.Request) {
	var body domain.Currency
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Code) == "" {
		writeValidation(w, map[string][]string{"code": {"The code field is required."}})
		return
	}
	if body.Decimals == 0 && body.Code != "JPY" {
		body.Decimals = 2
	}
	c, err := s.currencies.Create(r.Context(), body)
	if err != nil {
		writeValidation(w, map[string][]string{"code": {"The code has already been taken."}})
		return
	}
	writeData(w, http.StatusCreated, c.JSON())
}

func (s *Server) currenciesShow(w http.ResponseWriter, r *http.Request) {
	c := s.currencyParam(w, r)
	if c == nil {
		return
	}
	writeData(w, http.StatusOK, c.JSON())
}

func (s *Server) currenciesUpdate(w http.ResponseWriter, r *http.Request) {
	cur := s.currencyParam(w, r)
	if cur == nil {
		return
	}
	body := *cur
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidation(w, map[string][]string{"code": {"The given data was invalid."}})
		return
	}
	c, err := s.currencies.Update(r.Context(), cur.ID, body)
	if err != nil {
		switch err.Error() {
		case "cannot unset base":
			writeMessage(w, 422, "Cannot unset base currency. Set another currency as base first.")
		case "base rate":
			writeMessage(w, 422, "Base currency rate must always be 1.")
		default:
			writeMessage(w, 422, err.Error())
		}
		return
	}
	writeData(w, http.StatusOK, c.JSON())
}

func (s *Server) currenciesDestroy(w http.ResponseWriter, r *http.Request) {
	cur := s.currencyParam(w, r)
	if cur == nil {
		return
	}
	if err := s.currencies.Delete(r.Context(), cur.ID); err != nil {
		switch err.Error() {
		case "in use":
			writeMessage(w, 422, "Cannot delete currency that is used by accounts.")
		case "base":
			writeMessage(w, 422, "Cannot delete base currency. Set another currency as base first.")
		case "last":
			writeMessage(w, 422, "Cannot delete the last currency.")
		default:
			writeMessage(w, 422, err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) currenciesSetBase(w http.ResponseWriter, r *http.Request) {
	cur := s.currencyParam(w, r)
	if cur == nil {
		return
	}
	c, err := s.currencies.SetBase(r.Context(), cur.ID)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusOK, c.JSON())
}

func (s *Server) currenciesConvert(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Amount float64 `json:"amount"`
		From   int64   `json:"from_currency_id"`
		To     int64   `json:"to_currency_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidation(w, map[string][]string{"amount": {"The given data was invalid."}})
		return
	}
	from, _ := s.currencies.ByID(r.Context(), body.From)
	to, _ := s.currencies.ByID(r.Context(), body.To)
	if from == nil || to == nil {
		writeValidation(w, map[string][]string{"from_currency_id": {"The selected currency is invalid."}})
		return
	}
	result := domain.Convert(body.Amount, *from, *to)
	writeJSON(w, http.StatusOK, map[string]any{
		"amount": body.Amount,
		"from":   from.JSON(),
		"to":     to.JSON(),
		"result": domainRound(result, to.Decimals),
	})
}

func (s *Server) accountsIndex(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("active") == "1" || r.URL.Query().Get("active") == "true"
	excludeDebts := r.URL.Query().Get("exclude_debts") == "1" || r.URL.Query().Get("exclude_debts") == "true"
	list, err := s.accounts.All(r.Context(), onlyActive, excludeDebts)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	payload := envelope{Data: mapSlice(list, domain.Account.JSON)}
	if r.URL.Query().Get("with_summary") == "1" || r.URL.Query().Get("with_summary") == "true" {
		base, _ := s.currencies.Base(r.Context())
		if id := r.URL.Query().Get("base_currency_id"); id != "" {
			if n, err := strconv.ParseInt(id, 10, 64); err == nil {
				base, _ = s.currencies.ByID(r.Context(), n)
			}
		}
		payload.Meta = nil
		writeJSON(w, http.StatusOK, map[string]any{
			"data":    payload.Data,
			"summary": s.accounts.Summary(r.Context(), base),
		})
		return
	}
	writeData(w, http.StatusOK, payload.Data)
}

func (s *Server) accountsStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name           string   `json:"name"`
		Type           string   `json:"type"`
		CurrencyID     *int64   `json:"currency_id"`
		CurrencyCode   string   `json:"currency_code"`
		InitialBalance float64  `json:"initial_balance"`
		IsActive       *bool    `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		writeValidation(w, map[string][]string{"name": {"The name field is required."}})
		return
	}
	var currencyID int64
	if body.CurrencyID != nil {
		currencyID = *body.CurrencyID
	} else if body.CurrencyCode != "" {
		c, err := s.currencies.FindOrCreateByCode(r.Context(), body.CurrencyCode)
		if err != nil || c == nil {
			writeValidation(w, map[string][]string{"currency_code": {"Unknown currency code."}})
			return
		}
		currencyID = c.ID
	} else {
		writeValidation(w, map[string][]string{"currency_id": {"The currency id field is required."}})
		return
	}
	active := true
	if body.IsActive != nil {
		active = *body.IsActive
	}
	a, err := s.accounts.Create(r.Context(), domain.Account{
		Name: body.Name, Type: body.Type, CurrencyID: currencyID,
		InitialBalance: body.InitialBalance, IsActive: active,
	})
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusCreated, a.JSON())
}

func (s *Server) accountsShow(w http.ResponseWriter, r *http.Request) {
	a := s.accountParam(w, r)
	if a == nil {
		return
	}
	writeData(w, http.StatusOK, a.JSON())
}

func (s *Server) accountsUpdate(w http.ResponseWriter, r *http.Request) {
	cur := s.accountParam(w, r)
	if cur == nil {
		return
	}
	var body struct {
		Name           *string  `json:"name"`
		Type           *string  `json:"type"`
		CurrencyID     *int64   `json:"currency_id"`
		CurrencyCode   *string  `json:"currency_code"`
		InitialBalance *float64 `json:"initial_balance"`
		IsActive       *bool    `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidation(w, map[string][]string{"name": {"The given data was invalid."}})
		return
	}
	if body.Name != nil {
		cur.Name = *body.Name
	}
	if body.Type != nil {
		cur.Type = *body.Type
	}
	if body.CurrencyID != nil {
		cur.CurrencyID = *body.CurrencyID
	}
	if body.CurrencyCode != nil && *body.CurrencyCode != "" {
		c, err := s.currencies.FindOrCreateByCode(r.Context(), *body.CurrencyCode)
		if err == nil && c != nil {
			cur.CurrencyID = c.ID
		}
	}
	if body.InitialBalance != nil {
		cur.InitialBalance = *body.InitialBalance
	}
	if body.IsActive != nil {
		cur.IsActive = *body.IsActive
	}
	a, err := s.accounts.Update(r.Context(), cur.ID, *cur)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusOK, a.JSON())
}

func (s *Server) accountsReorder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidation(w, map[string][]string{"ids": {"The ids field is required."}})
		return
	}
	if err := s.accounts.Reorder(r.Context(), body.IDs); err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) accountsDestroy(w http.ResponseWriter, r *http.Request) {
	a := s.accountParam(w, r)
	if a == nil {
		return
	}
	if err := s.accounts.Delete(r.Context(), a.ID); err != nil {
		writeMessage(w, 422, "Cannot delete account that has transactions.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) accountsBalanceHistory(w http.ResponseWriter, r *http.Request) {
	base, _ := s.currencies.Base(r.Context())
	start := r.URL.Query().Get("start_date")
	end := r.URL.Query().Get("end_date")
	if start == "" {
		start = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}
	if end == "" {
		end = time.Now().Format("2006-01-02")
	}
	if base == nil {
		writeJSON(w, http.StatusOK, map[string]any{"dates": []string{}, "series": []any{}, "currency": nil, "decimals": 2})
		return
	}
	accts, _ := s.accounts.All(r.Context(), true, true)
	dates := dateRange(start, end)
	series := []map[string]any{}
	for _, a := range accts {
		data := make([]float64, len(dates))
		native := make([]float64, len(dates))
		for i := range dates {
			native[i] = a.Balance
			if a.Currency != nil {
				data[i] = domain.Convert(a.Balance, *a.Currency, *base)
			}
		}
		code := ""
		if a.Currency != nil {
			code = a.Currency.Code
		}
		series = append(series, map[string]any{
			"id": a.ID, "name": a.Name, "type": a.Type, "data": data, "native_data": native, "currency": code,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"dates": dates, "series": series, "currency": base.Code, "decimals": base.Decimals,
	})
}

func (s *Server) accountsBalanceComparison(w http.ResponseWriter, r *http.Request) {
	base, _ := s.currencies.Base(r.Context())
	sum := s.accounts.Summary(r.Context(), base)
	writeJSON(w, http.StatusOK, map[string]any{
		"current":  sum["total_balance"],
		"previous": nil,
		"currency": sum["currency"],
		"decimals": sum["decimals"],
	})
}

func (s *Server) categoriesIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.categories.All(r.Context(), r.URL.Query().Get("type"))
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, mapSlice(list, domain.Category.JSON))
}

func (s *Server) categoriesStore(w http.ResponseWriter, r *http.Request) {
	var body domain.Category
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeValidation(w, map[string][]string{"name": {"The name field is required."}})
		return
	}
	c, err := s.categories.Create(r.Context(), body)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusCreated, c.JSON())
}

func (s *Server) categoriesShow(w http.ResponseWriter, r *http.Request) {
	c := s.categoryParam(w, r)
	if c == nil {
		return
	}
	writeData(w, http.StatusOK, c.JSON())
}

func (s *Server) categoriesUpdate(w http.ResponseWriter, r *http.Request) {
	cur := s.categoryParam(w, r)
	if cur == nil {
		return
	}
	body := *cur
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidation(w, map[string][]string{"name": {"The given data was invalid."}})
		return
	}
	c, err := s.categories.Update(r.Context(), cur.ID, body)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusOK, c.JSON())
}

func (s *Server) categoriesDestroy(w http.ResponseWriter, r *http.Request) {
	c := s.categoryParam(w, r)
	if c == nil {
		return
	}
	if err := s.categories.Delete(r.Context(), c.ID); err != nil {
		switch err.Error() {
		case "has transactions":
			writeMessage(w, 422, "Cannot delete category that has transactions.")
		case "last":
			writeMessage(w, 422, "Cannot delete the last category of this type.")
		default:
			writeMessage(w, 422, err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) categoriesStatistics(w http.ResponseWriter, r *http.Request) {
	c := s.categoryParam(w, r)
	if c == nil {
		return
	}
	stats, err := s.categories.Statistics(r.Context(), c.ID, r.URL.Query().Get("start_date"), r.URL.Query().Get("end_date"))
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) categoriesSummary(w http.ResponseWriter, r *http.Request) {
	typ := r.URL.Query().Get("type")
	if typ != "income" && typ != "expense" {
		writeValidation(w, map[string][]string{"type": {"The type field is required."}})
		return
	}
	list, _ := s.categories.All(r.Context(), typ)
	base, _ := s.currencies.Base(r.Context())
	code := ""
	if base != nil {
		code = base.Code
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":     mapSlice(list, domain.Category.JSON),
		"total":    0,
		"currency": code,
	})
}

func (s *Server) tagsIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.tags.All(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, mapSlice(list, domain.Tag.JSON))
}

func (s *Server) tagsStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		writeValidation(w, map[string][]string{"name": {"The name field is required."}})
		return
	}
	t, err := s.tags.Create(r.Context(), body.Name)
	if err != nil {
		writeValidation(w, map[string][]string{"name": {"The name has already been taken."}})
		return
	}
	writeData(w, http.StatusCreated, t.JSON())
}

func (s *Server) tagsShow(w http.ResponseWriter, r *http.Request) {
	t := s.tagParam(w, r)
	if t == nil {
		return
	}
	writeData(w, http.StatusOK, t.JSON())
}

func (s *Server) tagsUpdate(w http.ResponseWriter, r *http.Request) {
	cur := s.tagParam(w, r)
	if cur == nil {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeValidation(w, map[string][]string{"name": {"The name field is required."}})
		return
	}
	t, err := s.tags.Update(r.Context(), cur.ID, body.Name)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusOK, t.JSON())
}

func (s *Server) tagsDestroy(w http.ResponseWriter, r *http.Request) {
	t := s.tagParam(w, r)
	if t == nil {
		return
	}
	_ = s.tags.Delete(r.Context(), t.ID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) currencyParam(w http.ResponseWriter, r *http.Request) *domain.Currency {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	c, _ := s.currencies.ByID(r.Context(), id)
	if c == nil {
		writeMessage(w, http.StatusNotFound, "Not found.")
	}
	return c
}

func (s *Server) accountParam(w http.ResponseWriter, r *http.Request) *domain.Account {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	a, _ := s.accounts.ByID(r.Context(), id)
	if a == nil {
		writeMessage(w, http.StatusNotFound, "Not found.")
	}
	return a
}

func (s *Server) categoryParam(w http.ResponseWriter, r *http.Request) *domain.Category {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	c, _ := s.categories.ByID(r.Context(), id)
	if c == nil {
		writeMessage(w, http.StatusNotFound, "Not found.")
	}
	return c
}

func (s *Server) tagParam(w http.ResponseWriter, r *http.Request) *domain.Tag {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	t, _ := s.tags.ByID(r.Context(), id)
	if t == nil {
		writeMessage(w, http.StatusNotFound, "Not found.")
	}
	return t
}

func mapSlice[T any](in []T, fn func(T) map[string]any) []any {
	out := make([]any, 0, len(in))
	for _, v := range in {
		out = append(out, fn(v))
	}
	return out
}

func domainRound(v float64, decimals int) float64 {
	p := 1.0
	for i := 0; i < decimals; i++ {
		p *= 10
	}
	return float64(int(v*p+0.5)) / p
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
