package dto_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/auth"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/importer"
	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
)

var usd = money.Unit{ID: 1, Decimals: 2}

func marshal(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func decode(t *testing.T, v any) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(marshal(t, v)), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestDecimalMarshalingIsNotGlobal(t *testing.T) {
	// Importing dto must not change how a bare decimal marshals elsewhere.
	if got := marshal(t, decimal.RequireFromString("1.5")); got != `"1.5"` {
		t.Fatalf("bare decimal = %s", got)
	}
	if got := marshal(t, dto.Number(decimal.RequireFromString("1.5"))); got != `1.5` {
		t.Fatalf("Number = %s", got)
	}
	item := dto.NewTxItem(ledger.TxItem{Name: "x", Quantity: decimal.RequireFromString("2.50"), PricePerUnit: money.New(100, usd), TotalPrice: money.New(250, usd)})
	m := decode(t, item)
	if m["quantity"] != 2.5 || m["pricePerUnit"] != 1.0 || m["totalPrice"] != 2.5 {
		t.Fatalf("item = %v", m)
	}
}

func TestNewDebt(t *testing.T) {
	target := money.New(10000, usd)
	cur := &ledger.Currency{ID: 1, Code: "USD", Decimals: 2}
	d := dto.NewDebt(ledger.Account{Currency: cur, Balance: money.New(2500, usd), TargetAmount: &target})
	if d.PaymentProgress != 75 || d.TargetAmount.Minor() != 10000 {
		t.Fatalf("debt = %+v", d)
	}
	over := dto.NewDebt(ledger.Account{Currency: cur, Balance: money.New(-500, usd), TargetAmount: &target})
	if over.PaymentProgress != 100 {
		t.Fatalf("overpaid progress = %v, want 100", over.PaymentProgress)
	}
	// An account without a currency must not crash the handler.
	noCur := dto.NewDebt(ledger.Account{Balance: money.New(2500, usd)})
	if noCur.Currency != nil || noCur.PaymentProgress != 0 {
		t.Fatalf("debt without currency = %+v", noCur)
	}
}

func TestTransactionActionsFollowLedger(t *testing.T) {
	rid := int64(3)
	for _, tx := range []ledger.Transaction{
		{Type: "expense", Status: "confirmed"},
		{Type: "expense", Status: "pending"},
		{Type: "expense", Status: "pending", RecurringID: &rid},
		{Type: "expense", Status: "skipped", RecurringID: &rid},
		{Type: "transfer_in", Status: "confirmed"},
	} {
		want := tx.Allowed()
		got := dto.NewTransaction(tx).Actions
		if got.Edit != want.Edit || got.Delete != want.Delete || got.Duplicate != want.Duplicate || got.Confirm != want.Confirm || got.Skip != want.Skip {
			t.Errorf("%+v: actions %+v, ledger %+v", tx, got, want)
		}
	}
}

func TestSummaries(t *testing.T) {
	s := dto.NewTransactionSummary(ledger.TransactionSummary{Income: money.New(5000, usd), Expense: money.New(1250, usd), Count: 2})
	if s.Balance.Minor() != 3750 {
		t.Errorf("balance = %d", s.Balance.Minor())
	}
	d := dto.NewDebtSummary(ledger.DebtSummary{IOwe: money.New(3000, usd), OwedToMe: money.New(1000, usd)})
	if d.NetDebt.Minor() != -2000 {
		t.Errorf("net debt = %d", d.NetDebt.Minor())
	}
}

func TestAPITokenKeepsRequestKeys(t *testing.T) {
	exp := time.Date(2030, 1, 2, 3, 4, 5, 0, time.FixedZone("x", 3600))
	m := decode(t, dto.NewCreatedAPIToken(auth.APIToken{ID: 1, Name: "ci", AllSpaces: true, ExpiresAt: &exp}, "secret"))
	if m["all_spaces"] != true || m["token"] != "secret" || m["expires_at"] != "2030-01-02T02:04:05Z" ||
		m["last_used_at"] != nil || m["permissions"] == nil || m["space_ids"] == nil {
		t.Fatalf("token = %v", m)
	}
}

func TestWebAuthnCredTimes(t *testing.T) {
	at := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	got := marshal(t, dto.NewWebAuthnCred(auth.WebAuthnCred{ID: 1, CreatedAt: &at}))
	if want := `{"id":1,"name":null,"aaguid":null,"lastUsedAt":null,"createdAt":"2030-01-02T03:04:05Z"}`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestNewImport(t *testing.T) {
	im := importer.Import{
		ID: "i1", Status: "completed", CreatedCount: 2, SkippedCount: 1, ErrorCount: 1,
		Errors: []importer.RowError{{Row: 4, Message: "bad date"}},
		Meta: importer.Meta{
			Headers:         []string{"date", "amount"},
			DetectedFormats: importer.Formats{DateFormat: "ISO", AmountFormat: "US", HasHeader: true, Delimiter: ";"},
		},
	}
	m := decode(t, dto.NewImport(im))
	parse := m["parse"].(map[string]any)
	if f := parse["detectedFormats"].(map[string]any); f["delimiter"] != ";" || f["hasHeader"] != true || f["dateFormat"] != "ISO" {
		t.Errorf("formats = %v", f)
	}
	if len(parse["previewRows"].([]any)) != 0 {
		t.Errorf("previewRows = %v, want []", parse["previewRows"])
	}
	res := m["result"].(map[string]any)
	if errs := res["errors"].([]any); len(errs) != 1 || errs[0].(map[string]any)["message"] != "bad date" {
		t.Errorf("errors = %v", res["errors"])
	}
	if res["createdTags"] == nil || res["skippedDuplicates"] != 1.0 {
		t.Errorf("result = %v", res)
	}
}
