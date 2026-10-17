package dto_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
	"savvy-go/internal/refsync"
	"savvy-go/internal/spaces"
	"savvy-go/internal/transfers"
)

func TestNewSyncDiff(t *testing.T) {
	d := refsync.Diff{
		Categories: []refsync.CategoryConflict{{
			Type: "expense", Name: "Food",
			Present: []refsync.CategoryState{{Space: 2, IsDefault: true, Transactions: 3}},
		}},
		Tags:     []refsync.TagConflict{{Name: "trip", Present: []refsync.TagState{{Space: 2}}, Missing: []int64{1}}},
		Defaults: []refsync.DefaultConflict{{Type: "income", Defaults: map[int64]string{2: "Pay", 1: ""}}},
	}
	names := map[int64]string{1: "Home", 2: "Work"}
	got := decode(t, dto.NewSyncDiff(d, func(id int64) (string, bool) { n, ok := names[id]; return n, ok }))

	cat := got["categories"].([]any)[0].(map[string]any)
	if cat["missing"] == nil || len(cat["missing"].([]any)) != 0 {
		t.Errorf("missing = %v, want []", cat["missing"])
	}
	if p := cat["present"].([]any)[0].(map[string]any); p["spaceId"] != 2.0 || p["isDefault"] != true || p["icon"] != nil {
		t.Errorf("present = %v", p)
	}
	def := got["defaults"].([]any)[0].(map[string]any)["defaults"].([]any)
	if def[0].(map[string]any)["spaceId"] != 1.0 {
		t.Errorf("defaults are not in id order: %v", def)
	}
	spaces := got["spaces"].([]any)
	if len(spaces) != 2 || spaces[0].(map[string]any)["name"] != "Home" || spaces[1].(map[string]any)["name"] != "Work" {
		t.Errorf("spaces = %v", spaces)
	}
}

func TestNewTransferHidesTheOtherAccount(t *testing.T) {
	v := transfers.View{
		Transfer: transfers.Transfer{
			UUID: "u", Date: "2024-01-02", Version: 2,
			From: transfers.Side{AccountID: 10, Amount: 1250, Currency: "USD", Decimals: 2},
			To:   transfers.Side{AccountID: 20, Amount: 1000, Currency: "EUR", Decimals: 2},
		},
		Outgoing:   true,
		OtherSpace: &spaces.Space{ID: 9, Name: "Other"},
	}
	for _, c := range []struct {
		seesOther bool
		wantTo    bool
	}{{false, false}, {true, true}} {
		got := decode(t, dto.NewTransfer(v, c.seesOther))
		from, to := got["from"].(map[string]any), got["to"].(map[string]any)
		if from["accountId"] != 10.0 || from["amount"] != "12.5" {
			t.Errorf("from = %v", from)
		}
		if _, has := to["accountId"]; has != c.wantTo {
			t.Errorf("seesOther=%v: to = %v", c.seesOther, to)
		}
		if got["otherSpace"].(map[string]any)["name"] != "Other" {
			t.Errorf("otherSpace = %v", got["otherSpace"])
		}
		if _, has := got["remote"]; has {
			t.Errorf("remote present without a remote record")
		}
	}
}

func TestBalanceHistoryShape(t *testing.T) {
	id := int64(4)
	code := "USD"
	h := dto.BalanceHistory{
		Dates: []string{"2024-01-01"}, Currency: &code, Decimals: 2,
		Series: []dto.BalanceSeries{
			{Name: "Total", Type: "total", Data: []money.Money{money.New(100, usd)}, Currency: code},
			{ID: &id, Name: "Cash", Type: "cash", Data: []money.Money{money.New(100, usd)}, NativeData: []money.Money{money.New(100, usd)}, Currency: code},
		},
	}
	series := decode(t, h)["series"].([]any)
	total, cash := series[0].(map[string]any), series[1].(map[string]any)
	if total["id"] != nil || total["native_data"] != nil || cash["id"] != 4.0 || cash["native_data"] == nil {
		t.Errorf("series = %v", series)
	}
	if total["data"].([]any)[0] != 1.0 {
		t.Errorf("data = %v", total["data"])
	}
}

func TestTransactionPage(t *testing.T) {
	page := dto.TransactionPage{Data: []dto.Transaction{}, Meta: dto.PageMeta{CurrentPage: 1, LastPage: 3, PerPage: 25, Total: 60}}
	got := marshal(t, page)
	if want := `{"data":[],"meta":{"current_page":1,"last_page":3,"per_page":25,"total":60}}`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestInvitationAndMember(t *testing.T) {
	at := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	created := marshal(t, dto.InvitationCreated{ID: 3, EmailSent: true})
	if created != `{"id":3,"emailSent":true}` {
		t.Errorf("created = %s", created)
	}
	linked := marshal(t, dto.InvitationCreated{ID: 3, Token: "abc"})
	if linked != `{"id":3,"token":"abc","emailSent":false}` {
		t.Errorf("linked = %s", linked)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(marshal(t, dto.NewMember(spaces.Member{UserID: 1, CreatedAt: &at}))), &m); err != nil {
		t.Fatal(err)
	}
	if m["createdAt"] != "2030-01-02T03:04:05Z" || m["userId"] != 1.0 {
		t.Errorf("member = %v", m)
	}
}

func TestCatalogCurrencyRateIsANumber(t *testing.T) {
	got := marshal(t, dto.NewCatalogCurrency(ledger.Currency{Code: "EUR", Name: "Euro", Symbol: "€", Decimals: 2, Rate: decimal.RequireFromString("1.1")}))
	if want := `{"code":"EUR","name":"Euro","symbol":"€","decimals":2,"rate":1.1}`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
