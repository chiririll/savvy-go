// Package creditcards serves /credit-cards of a space.
package creditcards

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/ledger"
)

// today is the current date in the server's location.
func (s Handler) today() time.Time {
	n := time.Now().In(s.Loc)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

// card converts a card with the state it is in today.
func (s Handler) card(r *http.Request, c ledger.Card) (dto.CreditCard, error) {
	st, err := httpx.Scope(r).CreditCards.State(r.Context(), c, s.today())
	if err != nil {
		return dto.CreditCard{}, err
	}
	return dto.NewCreditCard(c, st), nil
}

func (s Handler) write(w http.ResponseWriter, r *http.Request, status int, c ledger.Card) {
	out, err := s.card(r, c)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, status, out)
}

func (s Handler) index(w http.ResponseWriter, r *http.Request) {
	active := r.URL.Query().Get("active") == "1" || r.URL.Query().Get("active") == "true"
	list, err := httpx.Scope(r).CreditCards.All(r.Context(), active)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]dto.CreditCard, len(list))
	for i, c := range list {
		if out[i], err = s.card(r, c); err != nil {
			httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	httpx.WriteData(w, http.StatusOK, out)
}

// terms are the billing terms in a request. On update what is left out keeps its
// value.
type terms struct {
	Limit        *decimal.Decimal `json:"limit"`
	Cycle        *string          `json:"cycle"`
	StatementDay *int             `json:"statement_day"`
	GraceDays    *int             `json:"grace_days"`
}

// over merges the request over the terms in cur.
func (t terms) over(cur ledger.CardInput) ledger.CardInput {
	if t.Limit != nil {
		cur.Limit = *t.Limit
	}
	if t.Cycle != nil {
		cur.Cycle = *t.Cycle
	}
	if t.StatementDay != nil {
		cur.StatementDay = *t.StatementDay
	}
	if t.GraceDays != nil {
		cur.GraceDays = *t.GraceDays
	}
	return cur
}

func (s Handler) store(w http.ResponseWriter, r *http.Request) {
	var body struct {
		terms
		AccountID      int64           `json:"account_id"`
		Name           string          `json:"name"`
		CurrencyID     int64           `json:"currency_id"`
		CurrencyCode   string          `json:"currency_code"`
		InitialBalance decimal.Decimal `json:"initial_balance"`
		IsActive       *bool           `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteValidation(w, map[string][]string{"limit": {"The given data was invalid."}})
		return
	}
	in := body.over(ledger.CardInput{})
	if body.AccountID != 0 { // an existing bank account becomes a card
		c, err := httpx.Scope(r).CreditCards.Convert(r.Context(), body.AccountID, in)
		switch {
		case err == nil && c == nil:
			httpx.WriteMessage(w, http.StatusNotFound, "Account not found.")
		case err != nil:
			writeError(w, err)
		default:
			s.write(w, r, http.StatusOK, *c)
		}
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		httpx.WriteValidation(w, map[string][]string{"name": {"The name field is required."}})
		return
	}
	currencyID, ok := s.currency(w, r, body.CurrencyID, body.CurrencyCode)
	if !ok {
		return
	}
	active := true
	if body.IsActive != nil {
		active = *body.IsActive
	}
	c, err := httpx.Scope(r).CreditCards.Create(r.Context(), ledger.NewCard{
		Name: body.Name, CurrencyID: currencyID, InitialBalance: body.InitialBalance, IsActive: active, Terms: in,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	s.write(w, r, http.StatusCreated, *c)
}

// currency finds the currency of a new card by id or code.
func (s Handler) currency(w http.ResponseWriter, r *http.Request, id int64, code string) (int64, bool) {
	if id != 0 {
		return id, true
	}
	if code != "" {
		cur, err := httpx.Scope(r).Currencies.EnsureByCode(r.Context(), code)
		if err != nil || cur == nil {
			httpx.WriteValidation(w, map[string][]string{"currency_code": {"Unknown currency code."}})
			return 0, false
		}
		return cur.ID, true
	}
	httpx.WriteValidation(w, map[string][]string{"currency_id": {"The currency id field is required."}})
	return 0, false
}

func (s Handler) show(w http.ResponseWriter, r *http.Request) {
	if c := param(w, r); c != nil {
		s.write(w, r, http.StatusOK, *c)
	}
}

func (s Handler) update(w http.ResponseWriter, r *http.Request) {
	cur := param(w, r)
	if cur == nil {
		return
	}
	// Decoded over the stored card (fields the body leaves out keep their values).
	edit := cur.Edit()
	var body struct {
		terms
		Name           *string          `json:"name"`
		InitialBalance *decimal.Decimal `json:"initial_balance"`
		IsActive       *bool            `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteValidation(w, map[string][]string{"name": {"The given data was invalid."}})
		return
	}
	if body.Name != nil {
		edit.Name = *body.Name
	}
	if body.InitialBalance != nil {
		edit.InitialBalance = *body.InitialBalance
	}
	if body.IsActive != nil {
		edit.IsActive = *body.IsActive
	}
	edit.Terms = body.over(edit.Terms)
	c, err := httpx.Scope(r).CreditCards.Update(r.Context(), cur.ID, edit)
	if err != nil {
		writeError(w, err)
		return
	}
	s.write(w, r, http.StatusOK, *c)
}

func (s Handler) destroy(w http.ResponseWriter, r *http.Request) {
	c := param(w, r)
	if c == nil {
		return
	}
	if err := httpx.Scope(r).CreditCards.Delete(r.Context(), c.ID); err != nil {
		httpx.WriteMessage(w, 422, "Cannot delete a card that has transactions.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func param(w http.ResponseWriter, r *http.Request) *ledger.Card {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	c, _ := httpx.Scope(r).CreditCards.ByID(r.Context(), id)
	if c == nil {
		httpx.WriteMessage(w, http.StatusNotFound, "Not a credit card.")
		return nil
	}
	return c
}

// writeError answers a failed create, convert or update: a bad term is a
// validation error on its key.
func writeError(w http.ResponseWriter, err error) {
	field := ""
	switch {
	case errors.Is(err, ledger.ErrCardLimit):
		field = "limit"
	case errors.Is(err, ledger.ErrCardDay):
		field = "statement_day"
	case errors.Is(err, ledger.ErrCardGrace):
		field = "grace_days"
	case errors.Is(err, ledger.ErrCardCycle):
		field = "cycle"
	case errors.Is(err, ledger.ErrCardNotConvertible):
		field = "account_id"
	}
	if field == "" {
		httpx.WriteAccountError(w, err)
		return
	}
	httpx.WriteValidation(w, map[string][]string{field: {err.Error()}})
}
