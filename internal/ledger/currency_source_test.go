package ledger

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

type fakeSource struct {
	catalog []Currency
	rates   map[string]decimal.Decimal
	err     error
	asked   []string // the base of each Rates call
}

func (f *fakeSource) Catalog(_ context.Context, base string) []Currency { return f.catalog }

func (f *fakeSource) Rates(_ context.Context, base string) (map[string]decimal.Decimal, error) {
	f.asked = append(f.asked, base)
	return f.rates, f.err
}

func TestUpdateRatesSetsEveryNonBaseCurrency(t *testing.T) {
	e := newMoneyEnv(t) // USD (base), JPY at 0.01, BTC at 50000
	src := &fakeSource{rates: map[string]decimal.Decimal{
		"USD": dec("1"), "JPY": dec("0.0066"), "BTC": dec("65000.5"), "EUR": dec("1.1"),
	}}
	curs := Currencies{DB: e.db, Source: src}
	updated, skipped, err := curs.UpdateRates(e.ctx)
	if err != nil || updated != 2 || skipped != 0 {
		t.Fatalf("updated %d skipped %d err %v", updated, skipped, err)
	}
	if len(src.asked) != 1 || src.asked[0] != "USD" {
		t.Fatalf("rates asked for %v, want the space's base", src.asked)
	}
	for code, want := range map[string]string{"USD": "1", "JPY": "0.0066", "BTC": "65000.5"} {
		c, _ := curs.ByCode(e.ctx, code)
		if !c.Rate.Equal(dec(want)) {
			t.Errorf("%s rate = %s, want %s", code, c.Rate, want)
		}
	}
	if eur, _ := curs.ByCode(e.ctx, "EUR"); eur != nil {
		t.Error("a currency the space does not use must not be created")
	}
}

func TestUpdateRatesSkipsCurrenciesWithoutAValidRate(t *testing.T) {
	e := newMoneyEnv(t)
	src := &fakeSource{rates: map[string]decimal.Decimal{"JPY": dec("0.0066"), "BTC": decimal.Zero}}
	curs := Currencies{DB: e.db, Source: src}
	updated, skipped, err := curs.UpdateRates(e.ctx)
	if err != nil || updated != 1 || skipped != 1 {
		t.Fatalf("updated %d skipped %d err %v", updated, skipped, err)
	}
	if btc, _ := curs.ByCode(e.ctx, "BTC"); !btc.Rate.Equal(dec("50000")) {
		t.Fatalf("BTC kept %s, want its old rate", btc.Rate)
	}
}

func TestUpdateRatesFailsWithoutLiveRates(t *testing.T) {
	e := newMoneyEnv(t)
	down := errors.New("down")
	for name, curs := range map[string]Currencies{
		"no source":          {DB: e.db},
		"source unavailable": {DB: e.db, Source: &fakeSource{err: down}},
	} {
		if _, _, err := curs.UpdateRates(e.ctx); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if jpy, _ := (Currencies{DB: e.db}).ByCode(e.ctx, "JPY"); !jpy.Rate.Equal(dec("0.01")) {
		t.Fatalf("a failed update changed JPY to %s", jpy.Rate)
	}
}

func TestCatalogOffersOnlyCurrenciesTheSpaceLacks(t *testing.T) {
	e := newMoneyEnv(t)
	src := &fakeSource{catalog: []Currency{
		{Code: "USD", Name: "US Dollar", Symbol: "$", Decimals: 2, Rate: dec("1")},
		{Code: "EUR", Name: "Euro", Symbol: "€", Decimals: 2, Rate: dec("1.1")},
	}}
	list := (Currencies{DB: e.db, Source: src}).Catalog(e.ctx)
	if len(list) != 1 || list[0]["code"] != "EUR" {
		t.Fatalf("catalog = %v", list)
	}
	if rate, _ := list[0]["rate"].(decimal.Decimal); !rate.Equal(dec("1.1")) {
		t.Fatalf("rate = %v", list[0]["rate"])
	}
	if got := (Currencies{DB: e.db}).Catalog(e.ctx); len(got) != 0 {
		t.Fatalf("without a source the catalog is empty, got %v", got)
	}
}

func TestFindOrCreateByCodeTakesTheCatalogRate(t *testing.T) {
	e := newMoneyEnv(t)
	src := &fakeSource{catalog: []Currency{{Code: "EUR", Name: "Euro", Symbol: "€", Decimals: 2, Rate: dec("1.1")}}}
	curs := Currencies{DB: e.db, Source: src}
	got, err := curs.FindOrCreateByCode(e.ctx, "eur")
	if err != nil || got.Code != "EUR" || !got.Rate.Equal(dec("1.1")) || got.IsBase {
		t.Fatalf("created %+v, %v", got, err)
	}
	again, err := curs.FindOrCreateByCode(e.ctx, "EUR")
	if err != nil || again.ID != got.ID {
		t.Fatalf("a second call must find it: %+v %v", again, err)
	}
	if _, err := curs.FindOrCreateByCode(e.ctx, "XXX"); err == nil {
		t.Fatal("a code the catalog does not have must be refused")
	}
}
