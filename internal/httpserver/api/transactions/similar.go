package transactions

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/shopspring/decimal"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/ledger"
)

// similarItem is one thing to look for: what a bank line or receipt says.
type similarItem struct {
	Key         string           `json:"key"`
	ExternalID  *string          `json:"external_id"`
	Date        string           `json:"date"`
	Amount      *decimal.Decimal `json:"amount"`
	Currency    string           `json:"currency"`
	AccountID   int64            `json:"account_id"`
	Description *string          `json:"description"`
}

// transactionsFindSimilar answers, for each item, which entered transactions
// look like it. It stores nothing.
func transactionsFindSimilar(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Items         []similarItem `json:"items"`
		ToleranceDays *int          `json:"date_tolerance_days"`
		MaxPerItem    int           `json:"max_per_item"`
	}
	data, err := io.ReadAll(r.Body)
	if err == nil {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		err = dec.Decode(&body)
	}
	if err != nil {
		httpx.WriteMessage(w, 422, "The given data was invalid: "+err.Error())
		return
	}
	if len(body.Items) == 0 || len(body.Items) > maxBulk {
		httpx.WriteMessage(w, 422, "Send between 1 and "+strconv.Itoa(maxBulk)+" items.")
		return
	}
	opt := ledger.SimilarOptions{ToleranceDays: 1, MaxPerItem: body.MaxPerItem}
	if body.ToleranceDays != nil {
		opt.ToleranceDays = *body.ToleranceDays
	}
	qs := make([]ledger.SimilarQuery, len(body.Items))
	for i, it := range body.Items {
		qs[i] = ledger.SimilarQuery{
			Key: it.Key, ExternalID: it.ExternalID, Date: it.Date, Currency: it.Currency,
			AccountID: it.AccountID, Description: it.Description,
		}
		if it.Amount != nil {
			qs[i].Amount = *it.Amount
		}
	}
	found, err := httpx.Scope(r).Txs.FindSimilar(r.Context(), qs, opt)
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	out := make([]dto.SimilarResult, len(found))
	for i, ms := range found {
		out[i] = dto.NewSimilarResult(i, body.Items[i].Key, ms)
	}
	httpx.WriteData(w, http.StatusOK, out)
}
