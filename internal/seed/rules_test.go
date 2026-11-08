package seed

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
)

// A template is read by clients as it is stored, so a transfer without a fee
// has empty lists, not null.
func TestTemplateHasNoNullLists(t *testing.T) {
	out := decimal.NewFromInt(-800)
	raw, err := templateJSON(ledger.TxInput{
		Type:     "transfer",
		Payments: []ledger.PaymentInput{{AccountID: 1, Amount: &out}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"payments", "lines", "tag_ids"} {
		if list, ok := got[key].([]any); !ok || list == nil {
			t.Errorf("%s = %v, want a list", key, got[key])
		}
	}
}
