// Package transactions serves /transactions of a space.
package transactions

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"savvy-go/internal/httpserver/httpx"
	"strconv"
	"strings"

	"savvy-go/internal/db/filter"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/ledger"
	"savvy-go/internal/store"
)

func transactionsIndex(w http.ResponseWriter, r *http.Request) {
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
	q := r.URL.Query()
	f.SortBy, f.SortDir = q.Get("sort_by"), q.Get("sort_direction")
	f.CategoryIDs = int64List(q, "category_ids")
	f.TagIDs = int64List(q, "tag_ids")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	per, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	list, total, err := httpx.Scope(r).Txs.Filtered(r.Context(), f, page, per)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
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
	payload := dto.TransactionPage{
		Data: dto.Map(list, dto.NewTransaction),
		Meta: dto.PageMeta{CurrentPage: page, LastPage: last, PerPage: per, Total: total},
	}
	if r.URL.Query().Get("with_summary") == "1" || r.URL.Query().Get("with_summary") == "true" {
		summary := dto.NewTransactionSummary(httpx.Scope(r).Txs.Summary(r.Context(), false))
		payload.Summary = &summary
	}
	httpx.WriteJSON(w, http.StatusOK, payload)
}

func int64List(q url.Values, key string) []int64 {
	var out []int64
	for _, raw := range append(q[key], q[key+"[]"]...) {
		for _, part := range strings.Split(raw, ",") {
			if id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil && id > 0 {
				out = append(out, id)
			}
		}
	}
	return out
}

func transactionsStore(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeTx(w, r)
	if !ok {
		return
	}
	tx, err := httpx.Scope(r).Txs.Create(r.Context(), in)
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	if tx.Status == "confirmed" {
		httpx.Scope(r).Automation.Process(r.Context(), "on_transaction_create", tx)
		if fresh, e := httpx.Scope(r).Txs.ByID(r.Context(), tx.ID); e == nil && fresh != nil {
			fresh.Difference = tx.Difference // only a save knows it
			tx = fresh
		}
	}
	httpx.WriteData(w, http.StatusCreated, dto.NewTransaction(*tx))
}

func transactionsShow(w http.ResponseWriter, r *http.Request) {
	tx := txParam(w, r)
	if tx == nil {
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewTransaction(*tx))
}

func transactionsUpdate(w http.ResponseWriter, r *http.Request) {
	cur := txParam(w, r)
	if cur == nil {
		return
	}
	in, ok := decodeTx(w, r)
	if !ok {
		return
	}
	tx, err := httpx.Scope(r).Txs.Update(r.Context(), cur.ID, in)
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	if tx.Status == "confirmed" {
		httpx.Scope(r).Automation.Process(r.Context(), "on_transaction_update", tx)
		if fresh, e := httpx.Scope(r).Txs.ByID(r.Context(), tx.ID); e == nil && fresh != nil {
			fresh.Difference = tx.Difference // only a save knows it
			tx = fresh
		}
	}
	httpx.WriteData(w, http.StatusOK, dto.NewTransaction(*tx))
}

func transactionsDestroy(w http.ResponseWriter, r *http.Request) {
	tx := txParam(w, r)
	if tx == nil {
		return
	}
	if err := httpx.Scope(r).Txs.Delete(r.Context(), tx.ID); err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// settle runs fn, which confirms or skips an occurrence, and makes the next
// occurrence of its rule in the same database transaction, so a crash between
// the two leaves neither.
func settle(r *http.Request, id int64, fn func(ledger.Transactions) (*ledger.Transaction, error)) (*ledger.Transaction, error) {
	scope := httpx.Scope(r)
	var out *ledger.Transaction
	err := store.Tx(r.Context(), scope.DB, func(d store.DB) error {
		var err error
		if out, err = fn(scope.Txs.On(d)); err != nil {
			return err
		}
		recur := scope.Recurring
		recur.DB = d // read the rule inside the same database transaction
		rule, err := recur.RuleOf(r.Context(), id)
		if err != nil || rule == 0 {
			return err
		}
		return recur.AdvanceAfterOccurrence(r.Context(), d, rule)
	})
	return out, err
}

func transactionsConfirm(w http.ResponseWriter, r *http.Request) {
	tx := txParam(w, r)
	if tx == nil {
		return
	}
	var body struct {
		Date     *string               `json:"date"`
		Payments []ledger.PaymentInput `json:"payments"`
		Lines    []ledger.LineInput    `json:"lines"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		httpx.WriteValidation(w, map[string][]string{"body": {"The given data was invalid."}})
		return
	}
	date := ""
	if body.Date != nil {
		date = *body.Date
	}
	out, err := settle(r, tx.ID, func(txs ledger.Transactions) (*ledger.Transaction, error) {
		return txs.Confirm(r.Context(), tx.ID, date, &ledger.Amendment{Payments: body.Payments, Lines: body.Lines})
	})
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	httpx.Scope(r).Automation.Process(r.Context(), "on_transaction_create", out)
	if fresh, e := httpx.Scope(r).Txs.ByID(r.Context(), out.ID); e == nil && fresh != nil {
		out = fresh
	}
	httpx.WriteData(w, http.StatusOK, dto.NewTransaction(*out))
}

// transactionsReorder sets the order of transactions within one day. ids go
// oldest first, whichever way the client lists them.
func transactionsReorder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Date string  `json:"date"`
		IDs  []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Date == "" || len(body.IDs) == 0 {
		httpx.WriteValidation(w, map[string][]string{
			"ids": {"The date and ids fields are required."},
		})
		return
	}
	if err := httpx.Scope(r).Txs.Reorder(r.Context(), body.Date, body.IDs); err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	httpx.WriteSuccess(w)
}

func transactionsSkip(w http.ResponseWriter, r *http.Request) {
	tx := txParam(w, r)
	if tx == nil {
		return
	}
	out, err := settle(r, tx.ID, func(txs ledger.Transactions) (*ledger.Transaction, error) {
		return txs.Skip(r.Context(), tx.ID)
	})
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewTransaction(*out))
}

func transactionsDuplicate(w http.ResponseWriter, r *http.Request) {
	tx := txParam(w, r)
	if tx == nil {
		return
	}
	out, err := httpx.Scope(r).Txs.Duplicate(r.Context(), tx.ID)
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusCreated, dto.NewTransaction(*out))
}

func transactionsSummary(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, dto.NewTransactionSummary(httpx.Scope(r).Txs.Summary(r.Context(), false)))
}

func transactionsPendingSummary(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, dto.NewTransactionSummary(httpx.Scope(r).Txs.Summary(r.Context(), true)))
}

func txParam(w http.ResponseWriter, r *http.Request) *ledger.Transaction {
	return httpx.Lookup(w, r, httpx.Scope(r).Txs.ByID)
}

// decodeTx reads a transaction from the body: type, description, date,
// payments (account_id, amount), lines (category_id, currency_id, amount,
// note) and tag_ids. On update, what the body leaves out is kept. A key the
// API does not know is refused, so a client still sending the flat shape of
// older versions (account_id, amount, items) is told instead of ignored.
func decodeTx(w http.ResponseWriter, r *http.Request) (ledger.TxInput, bool) {
	data, err := io.ReadAll(r.Body)
	var in ledger.TxInput
	if err == nil {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		err = dec.Decode(&in)
	}
	if err != nil {
		httpx.WriteMessage(w, 422, "The given data was invalid: "+err.Error())
		return ledger.TxInput{}, false
	}
	in.Status = nil // set by the date and by confirm, not by the client
	// An explicit null clears the description; leaving it out keeps it.
	var sent map[string]json.RawMessage
	if json.Unmarshal(data, &sent) == nil && string(sent["description"]) == "null" {
		blank := ""
		in.Description = &blank
	}
	return in, true
}
