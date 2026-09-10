package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"savvy-go/internal/domain"

	"github.com/go-chi/chi/v5"
)

func (s *Server) debtsIndex(w http.ResponseWriter, r *http.Request) {
	include := r.URL.Query().Get("include_completed") == "1" || r.URL.Query().Get("include_completed") == "true"
	list, err := s.debts.All(r.Context(), include)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	payload := map[string]any{"data": mapSlice(list, domain.Account.DebtJSON)}
	if r.URL.Query().Get("with_summary") == "1" || r.URL.Query().Get("with_summary") == "true" {
		payload["summary"] = s.debts.Summary(r.Context())
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) debtsStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Origin       string  `json:"origin"`
		Name         string  `json:"name"`
		DebtType     string  `json:"debt_type"`
		AccountID    int64   `json:"account_id"`
		CurrencyID   int64   `json:"currency_id"`
		Amount       float64 `json:"amount"`
		Date         string  `json:"date"`
		DueDate      *string `json:"due_date"`
		Counterparty *string `json:"counterparty"`
		Description  *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeValidation(w, map[string][]string{"name": {"The name field is required."}})
		return
	}
	d, err := s.debts.Create(r.Context(), body.Name, body.DebtType, body.CurrencyID, body.AccountID, body.Amount, body.Date, body.Origin, body.DueDate, body.Counterparty, body.Description)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusCreated, d.DebtJSON())
}

func (s *Server) debtsShow(w http.ResponseWriter, r *http.Request) {
	d := s.debtParam(w, r)
	if d == nil {
		return
	}
	writeData(w, http.StatusOK, d.DebtJSON())
}

func (s *Server) debtsUpdate(w http.ResponseWriter, r *http.Request) {
	cur := s.debtParam(w, r)
	if cur == nil {
		return
	}
	var body struct {
		Name         *string  `json:"name"`
		DebtType     *string  `json:"debt_type"`
		CurrencyID   *int64   `json:"currency_id"`
		Amount       *float64 `json:"amount"`
		DueDate      *string  `json:"due_date"`
		Counterparty *string  `json:"counterparty"`
		Description  *string  `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidation(w, map[string][]string{"name": {"The given data was invalid."}})
		return
	}
	if body.Name != nil {
		cur.Name = *body.Name
	}
	if body.DebtType != nil {
		cur.DebtType = body.DebtType
	}
	if body.CurrencyID != nil {
		cur.CurrencyID = *body.CurrencyID
	}
	if body.Amount != nil {
		cur.TargetAmount = body.Amount
	}
	if body.DueDate != nil {
		cur.DueDate = body.DueDate
	}
	if body.Counterparty != nil {
		cur.Counterparty = body.Counterparty
	}
	if body.Description != nil {
		cur.DebtDesc = body.Description
	}
	updated, err := s.accounts.Update(r.Context(), cur.ID, *cur)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusOK, updated.DebtJSON())
}

func (s *Server) debtsDestroy(w http.ResponseWriter, r *http.Request) {
	d := s.debtParam(w, r)
	if d == nil {
		return
	}
	if err := s.debts.Delete(r.Context(), d.ID); err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) debtsPayment(w http.ResponseWriter, r *http.Request) {
	s.debtMove(w, r, false)
}

func (s *Server) debtsCollect(w http.ResponseWriter, r *http.Request) {
	s.debtMove(w, r, true)
}

func (s *Server) debtMove(w http.ResponseWriter, r *http.Request, collect bool) {
	d := s.debtParam(w, r)
	if d == nil {
		return
	}
	var body struct {
		AccountID   int64   `json:"account_id"`
		Amount      float64 `json:"amount"`
		Date        string  `json:"date"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidation(w, map[string][]string{"amount": {"The given data was invalid."}})
		return
	}
	tx, err := s.debts.Payment(r.Context(), d.ID, body.AccountID, body.Amount, body.Date, body.Description, collect)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusOK, tx.JSON())
}

func (s *Server) debtsReopen(w http.ResponseWriter, r *http.Request) {
	d := s.debtParam(w, r)
	if d == nil {
		return
	}
	out, err := s.debts.Reopen(r.Context(), d.ID)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusOK, out.DebtJSON())
}

func (s *Server) debtsSummary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.debts.Summary(r.Context()))
}

func (s *Server) debtParam(w http.ResponseWriter, r *http.Request) *domain.Account {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	a, _ := s.accounts.ByID(r.Context(), id)
	if a == nil || a.Type != "debt" {
		writeMessage(w, http.StatusNotFound, "Not a debt.")
		return nil
	}
	return a
}
