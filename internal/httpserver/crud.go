package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"savvy-go/internal/domain"
	"savvy-go/internal/httpserver/dto"
	"savvy-go/internal/money"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

func (s *Server) currenciesIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.currencies.All(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, dto.Map(list, dto.NewCurrency))
}

func (s *Server) currenciesCatalog(w http.ResponseWriter, r *http.Request) {
	writeData(w, http.StatusOK, s.currencies.Catalog(r.Context()))
}

func (s *Server) currenciesStore(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	var body domain.Currency
	var given struct {
		Decimals *int `json:"decimals"`
	}
	if err := json.Unmarshal(raw, &body); err != nil || strings.TrimSpace(body.Code) == "" {
		writeValidation(w, map[string][]string{"code": {"The code field is required."}})
		return
	}
	_ = json.Unmarshal(raw, &given)
	if given.Decimals == nil { // absent, not zero: zero decimals is a real choice
		body.Decimals = 2
		if strings.EqualFold(body.Code, "JPY") {
			body.Decimals = 0
		}
	}
	c, err := s.currencies.Create(r.Context(), body)
	switch {
	case errors.Is(err, domain.ErrInvalidDecimals):
		writeValidation(w, map[string][]string{"decimals": {err.Error()}})
		return
	case errors.Is(err, domain.ErrInvalidRate):
		writeValidation(w, map[string][]string{"rate": {err.Error()}})
		return
	case err != nil:
		writeValidation(w, map[string][]string{"code": {"The code has already been taken."}})
		return
	}
	writeData(w, http.StatusCreated, dto.NewCurrency(*c))
}

func (s *Server) currenciesShow(w http.ResponseWriter, r *http.Request) {
	c := s.currencyParam(w, r)
	if c == nil {
		return
	}
	writeData(w, http.StatusOK, dto.NewCurrency(*c))
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
	if errors.Is(err, domain.ErrDecimalsImmutable) {
		writeMessage(w, 422, "Currency decimals cannot be changed after creation.")
		return
	}
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
	writeData(w, http.StatusOK, dto.NewCurrency(*c))
}

func (s *Server) currenciesDestroy(w http.ResponseWriter, r *http.Request) {
	cur := s.currencyParam(w, r)
	if cur == nil {
		return
	}
	if err := s.currencies.Delete(r.Context(), cur.ID); err != nil {
		switch err.Error() {
		case "in use":
			writeMessage(w, 422, "Cannot delete currency that is used by accounts or budgets.")
		case "base":
			writeMessage(w, 422, "Cannot delete base currency. Set another currency as base first.")
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
	writeData(w, http.StatusOK, dto.NewCurrency(*c))
}

func (s *Server) currenciesConvert(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Amount decimal.Decimal `json:"amount"`
		From   int64           `json:"from_currency_id"`
		To     int64           `json:"to_currency_id"`
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
	amount, err := money.FromInput(body.Amount, from.Unit())
	var result money.Money
	if err == nil {
		result, err = domain.TryConvert(amount, *from, *to)
	}
	if err != nil {
		writeValidation(w, map[string][]string{"amount": {"The amount is out of range."}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"amount": amount,
		"from":   dto.NewCurrency(*from),
		"to":     dto.NewCurrency(*to),
		"result": result,
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
	payload := envelope{Data: dto.Map(list, dto.NewAccount)}
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
			"summary": dto.NewAccountsSummary(s.accounts.Summary(r.Context(), base)),
		})
		return
	}
	writeData(w, http.StatusOK, payload.Data)
}

func (s *Server) accountsStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name           string          `json:"name"`
		Type           string          `json:"type"`
		CurrencyID     *int64          `json:"currency_id"`
		CurrencyCode   string          `json:"currency_code"`
		InitialBalance decimal.Decimal `json:"initial_balance"`
		IsActive       *bool           `json:"is_active"`
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
	a, err := s.accounts.Create(r.Context(), domain.AccountInput{
		Name: body.Name, Type: body.Type, CurrencyID: currencyID,
		InitialBalance: body.InitialBalance, IsActive: active,
	})
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeData(w, http.StatusCreated, dto.NewAccount(*a))
}

func (s *Server) accountsShow(w http.ResponseWriter, r *http.Request) {
	a := s.accountParam(w, r)
	if a == nil {
		return
	}
	writeData(w, http.StatusOK, dto.NewAccount(*a))
}

func (s *Server) accountsUpdate(w http.ResponseWriter, r *http.Request) {
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
		writeValidation(w, map[string][]string{"name": {"The given data was invalid."}})
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
		c, err := s.currencies.ByCode(r.Context(), *body.CurrencyCode)
		if err != nil || c == nil || c.ID != in.CurrencyID {
			writeAccountError(w, domain.ErrAccountCurrencyImmutable)
			return
		}
	}
	if body.InitialBalance != nil {
		in.InitialBalance = *body.InitialBalance
	}
	if body.IsActive != nil {
		in.IsActive = *body.IsActive
	}
	a, err := s.accounts.Update(r.Context(), cur.ID, in)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeData(w, http.StatusOK, dto.NewAccount(*a))
}

// writeAccountError maps account create/update failures to responses.
func writeAccountError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrAccountCurrencyImmutable):
		writeMessage(w, 422, "Account currency cannot be changed after creation.")
	case errors.Is(err, domain.ErrUnknownCurrency):
		writeValidation(w, map[string][]string{"currency_id": {"The selected currency is invalid."}})
	default:
		writeMessage(w, 422, err.Error())
	}
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
	start, end := s.queryPeriod(r)
	if base == nil {
		writeJSON(w, http.StatusOK, map[string]any{"dates": []string{}, "series": []any{}, "currency": nil, "decimals": 2})
		return
	}
	accts, _ := s.accounts.All(r.Context(), true, true)
	dates := dateRange(start, end)
	series := []map[string]any{}
	total := make([]money.Money, len(dates))
	for i := range total {
		total[i] = money.Zero(base.Unit())
	}
	for _, a := range accts {
		native, err := s.accounts.BalanceSeries(r.Context(), a, dates) // in the account's own currency
		if err != nil {
			writeMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
		data := make([]money.Money, len(dates)) // in the base currency
		for i, bal := range native {
			data[i] = domain.Convert(bal, *a.Currency, *base)
			total[i] = total[i].Add(data[i])
		}
		series = append(series, map[string]any{
			"id": a.ID, "name": a.Name, "type": a.Type, "data": data, "native_data": native, "currency": a.Currency.Code,
		})
	}
	if len(accts) > 1 {
		series = append([]map[string]any{{"id": nil, "name": "Total", "type": "total", "data": total, "currency": base.Code}}, series...)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"dates": dates, "series": series, "currency": base.Code, "decimals": base.Decimals,
	})
}

// queryPeriod resolves start_date/end_date from the query, falling back to the
// shared default report period when either is missing or invalid.
func (s *Server) queryPeriod(r *http.Request) (start, end string) {
	q := r.URL.Query()
	rng := domain.ReportFilter{
		PeriodType: "custom",
		StartDate:  q.Get("start_date"),
		EndDate:    q.Get("end_date"),
	}.Range(time.Now().In(s.cfg.Location))
	return rng.Start.Format("2006-01-02"), rng.End.Format("2006-01-02")
}

// accountsBalanceComparison returns the current total balance and the total
// balance at the end of the day before the period starts.
func (s *Server) accountsBalanceComparison(w http.ResponseWriter, r *http.Request) {
	base, _ := s.currencies.Base(r.Context())
	sum := dto.NewAccountsSummary(s.accounts.Summary(r.Context(), base))
	var previous *money.Money
	if base != nil {
		start, _ := s.queryPeriod(r)
		if from, err := time.Parse("2006-01-02", start); err == nil {
			total := s.accounts.TotalAt(r.Context(), *base, from.AddDate(0, 0, -1).Format("2006-01-02"))
			previous = &total
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"current":  sum.TotalBalance,
		"previous": previous,
		"currency": sum.Currency,
		"decimals": sum.Decimals,
	})
}

func (s *Server) categoriesIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.categories.All(r.Context(), r.URL.Query().Get("type"))
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, dto.Map(list, dto.NewCategory))
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
	writeData(w, http.StatusCreated, dto.NewCategory(*c))
}

func (s *Server) categoriesShow(w http.ResponseWriter, r *http.Request) {
	c := s.categoryParam(w, r)
	if c == nil {
		return
	}
	writeData(w, http.StatusOK, dto.NewCategory(*c))
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
	writeData(w, http.StatusOK, dto.NewCategory(*c))
}

func (s *Server) categoriesDestroy(w http.ResponseWriter, r *http.Request) {
	c := s.categoryParam(w, r)
	if c == nil {
		return
	}
	var successorID *int64
	if raw := r.URL.Query().Get("successor_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			writeValidation(w, map[string][]string{"successor_id": {"The successor id is invalid."}})
			return
		}
		successorID = &id
	}
	if err := s.categories.Delete(r.Context(), c.ID, successorID); err != nil {
		switch err.Error() {
		case "default":
			writeMessage(w, 422, "Cannot delete the default category. Set another category as default first.")
		case "has transactions":
			writeMessage(w, 422, "Cannot delete category that has transactions.")
		case "invalid successor":
			writeMessage(w, 422, "The selected replacement category is invalid.")
		case "last":
			writeMessage(w, 422, "Cannot delete the last category of this type.")
		default:
			writeMessage(w, 422, err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) categoriesSetDefault(w http.ResponseWriter, r *http.Request) {
	c := s.categoryParam(w, r)
	if c == nil {
		return
	}
	updated, err := s.categories.SetDefault(r.Context(), c.ID)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusOK, dto.NewCategory(*updated))
}

func (s *Server) categoriesStatistics(w http.ResponseWriter, r *http.Request) {
	c := s.categoryParam(w, r)
	if c == nil {
		return
	}
	stats, err := s.categories.Statistics(r.Context(), c.ID, r.URL.Query().Get("start_date"), r.URL.Query().Get("end_date"))
	if err != nil || stats == nil {
		writeMessage(w, 422, "Could not compute category statistics.")
		return
	}
	writeJSON(w, http.StatusOK, dto.NewCategoryStatistics(*stats))
}

func (s *Server) categoriesSummary(w http.ResponseWriter, r *http.Request) {
	typ := r.URL.Query().Get("type")
	if typ != "income" && typ != "expense" {
		writeValidation(w, map[string][]string{"type": {"The type field is required."}})
		return
	}
	q := r.URL.Query()
	list, total := s.reports.CategorySummary(r.Context(), domain.ReportFilter{
		PeriodType: "custom",
		StartDate:  q.Get("start_date"),
		EndDate:    q.Get("end_date"),
	}, typ)
	base, _ := s.currencies.Base(r.Context())
	code := ""
	if base != nil {
		code = base.Code
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":     dto.Map(list, dto.NewCategory),
		"total":    total,
		"currency": code,
	})
}

func (s *Server) tagsIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.tags.All(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, dto.Map(list, dto.NewTag))
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
	writeData(w, http.StatusCreated, dto.NewTag(*t))
}

func (s *Server) tagsShow(w http.ResponseWriter, r *http.Request) {
	t := s.tagParam(w, r)
	if t == nil {
		return
	}
	writeData(w, http.StatusOK, dto.NewTag(*t))
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
	writeData(w, http.StatusOK, dto.NewTag(*t))
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
