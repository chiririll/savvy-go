// Package transactions serves /transactions of a space.
package transactions

import (
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"savvy-go/internal/httpserver/httpx"
	"strconv"
	"strings"

	"savvy-go/internal/db/filter"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/ledger"

	"github.com/shopspring/decimal"
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
	in, ok := decodeTx(w, r, nil)
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
	in, ok := decodeTx(w, r, cur)
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
		if errors.Is(err, ledger.ErrTransferRow) {
			httpx.WriteMessage(w, 422, err.Error())
			return
		}
		httpx.WriteMessage(w, 422, "Scheduled occurrences cannot be deleted. Skip or confirm them instead.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func transactionsConfirm(w http.ResponseWriter, r *http.Request) {
	tx := txParam(w, r)
	if tx == nil {
		return
	}
	var body struct {
		Date     *string          `json:"date"`
		Amount   *decimal.Decimal `json:"amount"`
		ToAmount *decimal.Decimal `json:"to_amount"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	date := ""
	if body.Date != nil {
		date = *body.Date
	}
	out, err := httpx.Scope(r).Txs.Confirm(r.Context(), tx.ID, date, body.Amount, body.ToAmount)
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	if out.RecurringID != nil {
		_ = httpx.Scope(r).Recurring.AdvanceAfterOccurrence(r.Context(), *out.RecurringID)
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
	out, err := httpx.Scope(r).Txs.Skip(r.Context(), tx.ID)
	if err != nil {
		if errors.Is(err, ledger.ErrTransferRow) {
			httpx.WriteMessage(w, 422, err.Error())
			return
		}
		httpx.WriteMessage(w, 422, "Only a pending scheduled transaction can be skipped.")
		return
	}
	if out.RecurringID != nil {
		_ = httpx.Scope(r).Recurring.AdvanceAfterOccurrence(r.Context(), *out.RecurringID)
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

// decodeTx reads a transaction from the body; on update, over base: fields
// the body leaves out keep their values. Items and tags left out are kept by the update itself.
func decodeTx(w http.ResponseWriter, r *http.Request, base *ledger.Transaction) (ledger.TxInput, bool) {
	var body struct {
		Type        string           `json:"type"`
		AccountID   int64            `json:"account_id"`
		ToAccountID *int64           `json:"to_account_id"`
		CategoryID  *int64           `json:"category_id"`
		Amount      decimal.Decimal  `json:"amount"`
		ToAmount    *decimal.Decimal `json:"to_amount"`
		IsEstimated bool             `json:"is_estimated"`
		Description *string          `json:"description"`
		Date        *string          `json:"date"`
		TagIDs      []int64          `json:"tag_ids"`
		Items       []struct {
			Name         string          `json:"name"`
			Quantity     decimal.Decimal `json:"quantity"`
			PricePerUnit decimal.Decimal `json:"price_per_unit"`
		} `json:"items"`
	}
	if base != nil {
		body.Type, body.AccountID, body.ToAccountID = base.Type, base.AccountID, httpx.Clone(base.ToAccountID)
		body.CategoryID, body.Amount = httpx.Clone(base.CategoryID), base.Amount.Decimal()
		if base.ToAmount != nil {
			toAmount := base.ToAmount.Decimal()
			body.ToAmount = &toAmount
		}
		body.IsEstimated, body.Description, body.Date = base.IsEstimated, httpx.Clone(
			base.Description,
		), httpx.Clone(
			base.Date,
		)
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.AccountID == 0 {
		httpx.WriteValidation(w, map[string][]string{"account_id": {"The account id field is required."}})
		return ledger.TxInput{}, false
	}
	in := ledger.TxInput{
		Type: body.Type, AccountID: body.AccountID, ToAccountID: body.ToAccountID,
		CategoryID: body.CategoryID, Amount: body.Amount, ToAmount: body.ToAmount,
		IsEstimated: body.IsEstimated, Description: body.Description, Date: body.Date,
		TagIDs: body.TagIDs,
	}
	for _, it := range body.Items {
		qty := it.Quantity
		if qty.IsZero() {
			qty = decimal.NewFromInt(1)
		}
		in.Items = append(in.Items, ledger.TxItemInput{Name: it.Name, Quantity: qty, PricePerUnit: it.PricePerUnit})
	}
	return in, true
}
