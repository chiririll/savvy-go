// Package currencies serves /currencies of a space.
package currencies

import (
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/apidocs"
	"savvy-go/internal/httpserver/dto"
	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
)

func currenciesIndex(w http.ResponseWriter, r *http.Request) {
	list, err := api.Scope(r).Currencies.All(r.Context())
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	api.WriteData(w, http.StatusOK, dto.Map(list, dto.NewCurrency))
}

func currenciesCatalog(w http.ResponseWriter, r *http.Request) {
	api.WriteData(w, http.StatusOK, api.Scope(r).Currencies.Catalog(r.Context()))
}

func currenciesStore(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	var body ledger.Currency
	var given struct {
		Decimals *int `json:"decimals"`
	}
	if err := json.Unmarshal(raw, &body); err != nil || strings.TrimSpace(body.Code) == "" {
		api.WriteValidation(w, map[string][]string{"code": {"The code field is required."}})
		return
	}
	_ = json.Unmarshal(raw, &given)
	if given.Decimals == nil { // absent, not zero: zero decimals is a real choice
		body.Decimals = 2
		if strings.EqualFold(body.Code, "JPY") {
			body.Decimals = 0
		}
	}
	c, err := api.Scope(r).Currencies.Create(r.Context(), body)
	switch {
	case errors.Is(err, ledger.ErrInvalidDecimals):
		api.WriteValidation(w, map[string][]string{"decimals": {err.Error()}})
		return
	case errors.Is(err, ledger.ErrInvalidRate):
		api.WriteValidation(w, map[string][]string{"rate": {err.Error()}})
		return
	case err != nil:
		api.WriteValidation(w, map[string][]string{"code": {"The code has already been taken."}})
		return
	}
	api.WriteData(w, http.StatusCreated, dto.NewCurrency(*c))
}

func currenciesShow(w http.ResponseWriter, r *http.Request) {
	c := currencyParam(w, r)
	if c == nil {
		return
	}
	api.WriteData(w, http.StatusOK, dto.NewCurrency(*c))
}

func currenciesUpdate(w http.ResponseWriter, r *http.Request) {
	cur := currencyParam(w, r)
	if cur == nil {
		return
	}
	body := *cur
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteValidation(w, map[string][]string{"code": {"The given data was invalid."}})
		return
	}
	c, err := api.Scope(r).Currencies.Update(r.Context(), cur.ID, body)
	if errors.Is(err, ledger.ErrDecimalsImmutable) {
		api.WriteMessage(w, 422, "Currency decimals cannot be changed after creation.")
		return
	}
	if err != nil {
		switch err.Error() {
		case "cannot unset base":
			api.WriteMessage(w, 422, "Cannot unset base currency. Set another currency as base first.")
		case "base rate":
			api.WriteMessage(w, 422, "Base currency rate must always be 1.")
		default:
			api.WriteMessage(w, 422, err.Error())
		}
		return
	}
	api.WriteData(w, http.StatusOK, dto.NewCurrency(*c))
}

func currenciesDestroy(w http.ResponseWriter, r *http.Request) {
	cur := currencyParam(w, r)
	if cur == nil {
		return
	}
	if err := api.Scope(r).Currencies.Delete(r.Context(), cur.ID); err != nil {
		switch err.Error() {
		case "in use":
			api.WriteMessage(w, 422, "Cannot delete currency that is used by accounts or budgets.")
		case "base":
			api.WriteMessage(w, 422, "Cannot delete base currency. Set another currency as base first.")
		default:
			api.WriteMessage(w, 422, err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func currenciesSetBase(w http.ResponseWriter, r *http.Request) {
	cur := currencyParam(w, r)
	if cur == nil {
		return
	}
	c, err := api.Scope(r).Currencies.SetBase(r.Context(), cur.ID)
	if err != nil {
		api.WriteMessage(w, 422, err.Error())
		return
	}
	api.WriteData(w, http.StatusOK, dto.NewCurrency(*c))
}

func currenciesConvert(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Amount decimal.Decimal `json:"amount"`
		From   int64           `json:"from_currency_id"`
		To     int64           `json:"to_currency_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteValidation(w, map[string][]string{"amount": {"The given data was invalid."}})
		return
	}
	from, _ := api.Scope(r).Currencies.ByID(r.Context(), body.From)
	to, _ := api.Scope(r).Currencies.ByID(r.Context(), body.To)
	if from == nil || to == nil {
		api.WriteValidation(w, map[string][]string{"from_currency_id": {"The selected currency is invalid."}})
		return
	}
	amount, err := money.FromInput(body.Amount, from.Unit())
	var result money.Money
	if err == nil {
		result, err = ledger.TryConvert(amount, *from, *to)
	}
	if err != nil {
		api.WriteValidation(w, map[string][]string{"amount": {"The amount is out of range."}})
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"amount": amount,
		"from":   dto.NewCurrency(*from),
		"to":     dto.NewCurrency(*to),
		"result": result,
	})
}

func currencyParam(w http.ResponseWriter, r *http.Request) *ledger.Currency {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	c, _ := api.Scope(r).Currencies.ByID(r.Context(), id)
	if c == nil {
		api.WriteMessage(w, http.StatusNotFound, "Not found.")
	}
	return c
}

//go:embed openapi.yaml
var spec []byte

func Routes(r chi.Router) {
	apidocs.Add(spec)
	r.Get("/currencies/catalog", currenciesCatalog)
	r.Get("/currencies", currenciesIndex)
	r.Post("/currencies", currenciesStore)
	r.Get("/currencies/{id}", currenciesShow)
	r.Put("/currencies/{id}", currenciesUpdate)
	r.Patch("/currencies/{id}", currenciesUpdate)
	r.Delete("/currencies/{id}", currenciesDestroy)
	r.Post("/currencies/{id}/set-base", currenciesSetBase)
	r.Post("/currencies/convert", currenciesConvert)
}
