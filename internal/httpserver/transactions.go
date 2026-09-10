package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"savvy-go/internal/db/filter"
	"savvy-go/internal/domain"

	"github.com/go-chi/chi/v5"
)

func (s *Server) transactionsIndex(w http.ResponseWriter, r *http.Request) {
	f := filter.TxFilter{
		Type: r.URL.Query().Get("type"), Status: r.URL.Query().Get("status"),
		StartDate: r.URL.Query().Get("start_date"), EndDate: r.URL.Query().Get("end_date"),
	}
	if v := r.URL.Query().Get("account_id"); v != "" {
		f.AccountID, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := r.URL.Query().Get("category_id"); v != "" {
		f.CategoryID, _ = strconv.ParseInt(v, 10, 64)
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	per, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	list, total, err := s.txs.Filtered(r.Context(), f, page, per)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if per <= 0 {
		per = 25
	}
	if page <= 0 {
		page = 1
	}
	last := (total + per - 1) / per
	if last < 1 {
		last = 1
	}
	payload := map[string]any{
		"data": mapSlice(list, domain.Transaction.JSON),
		"meta": map[string]any{
			"current_page": page, "last_page": last, "per_page": per, "total": total,
		},
	}
	if r.URL.Query().Get("with_summary") == "1" || r.URL.Query().Get("with_summary") == "true" {
		payload["summary"] = s.txs.Summary(r.Context(), false)
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) transactionsStore(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeTx(w, r)
	if !ok {
		return
	}
	tx, err := s.txs.Create(r.Context(), in)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	if tx.Status == "confirmed" {
		s.automation.Process(r.Context(), "on_transaction_create", tx)
		if fresh, e := s.txs.ByID(r.Context(), tx.ID); e == nil && fresh != nil {
			tx = fresh
		}
	}
	writeData(w, http.StatusCreated, tx.JSON())
}

func (s *Server) transactionsShow(w http.ResponseWriter, r *http.Request) {
	tx := s.txParam(w, r)
	if tx == nil {
		return
	}
	writeData(w, http.StatusOK, tx.JSON())
}

func (s *Server) transactionsUpdate(w http.ResponseWriter, r *http.Request) {
	cur := s.txParam(w, r)
	if cur == nil {
		return
	}
	in, ok := decodeTx(w, r)
	if !ok {
		return
	}
	if in.AccountID == 0 {
		in.AccountID = cur.AccountID
	}
	if in.Type == "" {
		in.Type = cur.Type
	}
	if in.Amount == 0 {
		in.Amount = cur.Amount
	}
	tx, err := s.txs.Update(r.Context(), cur.ID, in)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	if tx.Status == "confirmed" {
		s.automation.Process(r.Context(), "on_transaction_update", tx)
		if fresh, e := s.txs.ByID(r.Context(), tx.ID); e == nil && fresh != nil {
			tx = fresh
		}
	}
	writeData(w, http.StatusOK, tx.JSON())
}

func (s *Server) transactionsDestroy(w http.ResponseWriter, r *http.Request) {
	tx := s.txParam(w, r)
	if tx == nil {
		return
	}
	if err := s.txs.Delete(r.Context(), tx.ID); err != nil {
		writeMessage(w, 422, "Scheduled occurrences cannot be deleted. Skip or confirm them instead.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) transactionsConfirm(w http.ResponseWriter, r *http.Request) {
	tx := s.txParam(w, r)
	if tx == nil {
		return
	}
	var body struct {
		Date *string `json:"date"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	date := ""
	if body.Date != nil {
		date = *body.Date
	}
	out, err := s.txs.Confirm(r.Context(), tx.ID, date)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	if out.RecurringID != nil {
		_ = s.recurring.AdvanceAfterOccurrence(r.Context(), *out.RecurringID)
	}
	s.automation.Process(r.Context(), "on_transaction_create", out)
	if fresh, e := s.txs.ByID(r.Context(), out.ID); e == nil && fresh != nil {
		out = fresh
	}
	writeData(w, http.StatusOK, out.JSON())
}

func (s *Server) transactionsSkip(w http.ResponseWriter, r *http.Request) {
	tx := s.txParam(w, r)
	if tx == nil {
		return
	}
	out, err := s.txs.Skip(r.Context(), tx.ID)
	if err != nil {
		writeMessage(w, 422, "Only a pending scheduled transaction can be skipped.")
		return
	}
	if out.RecurringID != nil {
		_ = s.recurring.AdvanceAfterOccurrence(r.Context(), *out.RecurringID)
	}
	writeData(w, http.StatusOK, out.JSON())
}

func (s *Server) transactionsDuplicate(w http.ResponseWriter, r *http.Request) {
	tx := s.txParam(w, r)
	if tx == nil {
		return
	}
	out, err := s.txs.Duplicate(r.Context(), tx.ID)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusCreated, out.JSON())
}

func (s *Server) transactionsSummary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.txs.Summary(r.Context(), false))
}

func (s *Server) transactionsPendingSummary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.txs.Summary(r.Context(), true))
}

func (s *Server) txParam(w http.ResponseWriter, r *http.Request) *domain.Transaction {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	tx, _ := s.txs.ByID(r.Context(), id)
	if tx == nil {
		writeMessage(w, http.StatusNotFound, "Not found.")
	}
	return tx
}

func decodeTx(w http.ResponseWriter, r *http.Request) (domain.TxInput, bool) {
	var body struct {
		Type         string   `json:"type"`
		AccountID    int64    `json:"account_id"`
		ToAccountID  *int64   `json:"to_account_id"`
		CategoryID   *int64   `json:"category_id"`
		Amount       float64  `json:"amount"`
		ToAmount     *float64 `json:"to_amount"`
		ExchangeRate *float64 `json:"exchange_rate"`
		Description  *string  `json:"description"`
		Date         *string  `json:"date"`
		TagIDs       []int64  `json:"tag_ids"`
		Items        []struct {
			Name         string  `json:"name"`
			Quantity     float64 `json:"quantity"`
			PricePerUnit float64 `json:"price_per_unit"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.AccountID == 0 {
		writeValidation(w, map[string][]string{"account_id": {"The account id field is required."}})
		return domain.TxInput{}, false
	}
	in := domain.TxInput{
		Type: body.Type, AccountID: body.AccountID, ToAccountID: body.ToAccountID,
		CategoryID: body.CategoryID, Amount: body.Amount, ToAmount: body.ToAmount,
		ExchangeRate: body.ExchangeRate, Description: body.Description, Date: body.Date,
		TagIDs: body.TagIDs,
	}
	for _, it := range body.Items {
		qty := it.Quantity
		if qty == 0 {
			qty = 1
		}
		in.Items = append(in.Items, domain.TxItem{Name: it.Name, Quantity: qty, PricePerUnit: it.PricePerUnit})
	}
	return in, true
}
