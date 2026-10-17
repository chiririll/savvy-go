// Package debts serves /debts of a space.
package debts

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"savvy-go/internal/httpserver/httpx"
	"strconv"

	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/ledger"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

func debtsIndex(w http.ResponseWriter, r *http.Request) {
	include := r.URL.Query().Get("include_completed") == "1" || r.URL.Query().Get("include_completed") == "true"
	list, err := httpx.Scope(r).Debts.All(r.Context(), include)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	payload := dto.DebtList{Data: dto.Map(list, dto.NewDebt)}
	if r.URL.Query().Get("with_summary") == "1" || r.URL.Query().Get("with_summary") == "true" {
		summary := dto.NewDebtSummary(httpx.Scope(r).Debts.Summary(r.Context()))
		payload.Summary = &summary
	}
	httpx.WriteJSON(w, http.StatusOK, payload)
}

func debtsStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Origin       string          `json:"origin"`
		Name         string          `json:"name"`
		DebtType     string          `json:"debt_type"`
		AccountID    int64           `json:"account_id"`
		CurrencyID   int64           `json:"currency_id"`
		Amount       decimal.Decimal `json:"amount"`
		Date         string          `json:"date"`
		DueDate      *string         `json:"due_date"`
		Counterparty *string         `json:"counterparty"`
		Description  *string         `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		httpx.WriteValidation(w, map[string][]string{"name": {"The name field is required."}})
		return
	}
	d, err := httpx.Scope(r).Debts.Create(r.Context(), ledger.DebtInput{
		Name: body.Name, DebtType: body.DebtType, CurrencyID: body.CurrencyID, AccountID: body.AccountID,
		Amount: body.Amount, Date: body.Date, Origin: body.Origin,
		DueDate: body.DueDate, Counterparty: body.Counterparty, Description: body.Description,
	})
	if err != nil {
		httpx.WriteAccountError(w, err)
		return
	}
	httpx.WriteData(w, http.StatusCreated, dto.NewDebt(*d))
}

func debtsShow(w http.ResponseWriter, r *http.Request) {
	d := debtParam(w, r)
	if d == nil {
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewDebt(*d))
}

func debtsUpdate(w http.ResponseWriter, r *http.Request) {
	cur := debtParam(w, r)
	if cur == nil {
		return
	}
	// Decoded over the stored debt (see patch.go).
	in := cur.Input()
	var body struct {
		Name         string           `json:"name"`
		DebtType     *string          `json:"debt_type"`
		CurrencyID   int64            `json:"currency_id"`
		Amount       *decimal.Decimal `json:"amount"`
		DueDate      *string          `json:"due_date"`
		Counterparty *string          `json:"counterparty"`
		Description  *string          `json:"description"`
	}
	body.Name, body.DebtType, body.CurrencyID = in.Name, httpx.Clone(in.DebtType), in.CurrencyID
	body.Amount, body.DueDate = httpx.Clone(in.TargetAmount), httpx.Clone(in.DueDate)
	body.Counterparty, body.Description = httpx.Clone(in.Counterparty), httpx.Clone(in.DebtDesc)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteValidation(w, map[string][]string{"name": {"The given data was invalid."}})
		return
	}
	in.Name, in.DebtType, in.CurrencyID, in.TargetAmount = body.Name, body.DebtType, body.CurrencyID, body.Amount
	in.DueDate, in.Counterparty, in.DebtDesc = body.DueDate, body.Counterparty, body.Description
	updated, err := httpx.Scope(r).Accounts.Update(r.Context(), cur.ID, in)
	if err != nil {
		httpx.WriteAccountError(w, err)
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewDebt(*updated))
}

func debtsDestroy(w http.ResponseWriter, r *http.Request) {
	d := debtParam(w, r)
	if d == nil {
		return
	}
	if err := httpx.Scope(r).Debts.Delete(r.Context(), d.ID); err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func debtsPayment(w http.ResponseWriter, r *http.Request) {
	debtMove(w, r, false)
}

func debtsCollect(w http.ResponseWriter, r *http.Request) {
	debtMove(w, r, true)
}

func debtMove(w http.ResponseWriter, r *http.Request, collect bool) {
	d := debtParam(w, r)
	if d == nil {
		return
	}
	var body struct {
		AccountID   int64           `json:"account_id"`
		Amount      decimal.Decimal `json:"amount"`
		Date        string          `json:"date"`
		Description *string         `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteValidation(w, map[string][]string{"amount": {"The given data was invalid."}})
		return
	}
	tx, err := httpx.Scope(r).Debts.Payment(r.Context(), d.ID, body.AccountID, body.Amount, body.Date, body.Description, collect)
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewTransaction(*tx))
}

func debtsReopen(w http.ResponseWriter, r *http.Request) {
	d := debtParam(w, r)
	if d == nil {
		return
	}
	out, err := httpx.Scope(r).Debts.Reopen(r.Context(), d.ID)
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewDebt(*out))
}

func debtsSummary(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, dto.NewDebtSummary(httpx.Scope(r).Debts.Summary(r.Context())))
}

func debtParam(w http.ResponseWriter, r *http.Request) *ledger.Account {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	a, _ := httpx.Scope(r).Accounts.ByID(r.Context(), id)
	if a == nil || a.Type != "debt" {
		httpx.WriteMessage(w, http.StatusNotFound, "Not a debt.")
		return nil
	}
	return a
}
