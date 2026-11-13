// Package reconciliations serves /reconciliations of a space: the points
// that pin an account's balance to the real one on a date.
package reconciliations

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

func index(w http.ResponseWriter, r *http.Request) {
	f, ok := filter(w, r)
	if !ok {
		return
	}
	list(w, r, f)
}

func accountIndex(w http.ResponseWriter, r *http.Request) {
	acc := httpx.Lookup(w, r, httpx.Scope(r).Accounts.ByID)
	if acc == nil {
		return
	}
	f, ok := filter(w, r)
	if !ok {
		return
	}
	f.AccountIDs = []int64{acc.ID}
	list(w, r, f)
}

func list(w http.ResponseWriter, r *http.Request, f ledger.ReconciliationFilter) {
	views, err := httpx.Scope(r).Reconcile.List(r.Context(), f)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(views, dto.NewReconciliation))
}

func days(w http.ResponseWriter, r *http.Request) {
	f, ok := filter(w, r)
	if !ok {
		return
	}
	out, err := httpx.Scope(r).Reconcile.DayReconciliations(r.Context(), f.From, f.To)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(out, dto.NewReconciliationDay))
}

// filter reads account_ids (comma separated), start_date and end_date.
func filter(w http.ResponseWriter, r *http.Request) (ledger.ReconciliationFilter, bool) {
	q := r.URL.Query()
	f := ledger.ReconciliationFilter{From: q.Get("start_date"), To: q.Get("end_date")}
	for field, v := range map[string]string{"start_date": f.From, "end_date": f.To} {
		if _, err := time.Parse("2006-01-02", v); v != "" && err != nil {
			httpx.WriteValidation(w, map[string][]string{field: {"The " + field + " must be a day as YYYY-MM-DD."}})
			return f, false
		}
	}
	for _, raw := range append(q["account_ids"], q["account_ids[]"]...) {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.ParseInt(part, 10, 64)
			if err != nil || id <= 0 {
				httpx.WriteValidation(w, map[string][]string{"account_ids": {"The account_ids must be account ids."}})
				return f, false
			}
			f.AccountIDs = append(f.AccountIDs, id)
		}
	}
	return f, true
}

func store(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Date  string `json:"date"`
		Items []struct {
			AccountID     int64            `json:"accountId"`
			ActualBalance *decimal.Decimal `json:"actualBalance"`
			Note          *string          `json:"note"`
		} `json:"items"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.WriteValidation(w, map[string][]string{"items": {"The given data was invalid."}})
		return
	}
	items := make([]ledger.ReconcileItem, len(body.Items))
	for i, it := range body.Items {
		if it.ActualBalance == nil {
			httpx.WriteValidation(w, map[string][]string{
				"items." + strconv.Itoa(i) + ".actualBalance": {"The actualBalance field is required."},
			})
			return
		}
		items[i] = ledger.ReconcileItem{AccountID: it.AccountID, Actual: *it.ActualBalance, Note: it.Note}
	}
	views, err := httpx.Scope(r).Reconcile.Reconcile(r.Context(), body.Date, items, time.Now())
	if err != nil {
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusCreated, dto.Map(views, dto.NewReconciliation))
}

func destroy(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.WriteMessage(w, http.StatusNotFound, "Not found.")
		return
	}
	switch err := httpx.Scope(r).Reconcile.Delete(r.Context(), id); {
	case errors.Is(err, ledger.ErrUnknownReconciliation):
		httpx.WriteMessage(w, http.StatusNotFound, "Not found.")
	case err != nil:
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}
