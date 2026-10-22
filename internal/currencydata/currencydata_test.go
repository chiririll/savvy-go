package currencydata

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// fakeService serves the names file and one rates file per base, and counts
// the requests for each base.
type fakeService struct {
	srv  *httptest.Server
	mu   sync.Mutex
	hits map[string]int
	down bool
	// files holds the rates object of each base, by lower-case code.
	files map[string]string
}

func newFakeService(t *testing.T) *fakeService {
	t.Helper()
	f := &fakeService{
		hits: map[string]int{},
		files: map[string]string{
			"usd": `{"eur": 0.8, "jpy": 160, "btc": 0.00002, "zero": 0}`,
			"jpy": `{"usd": 0.00625, "eur": 0.005}`,
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/meta.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"USD": {"code": "USD", "name": "US Dollar", "symbol": "$", "symbol_native": "$", "decimal_digits": 2},
			"JPY": {"code": "JPY", "name": "Japanese Yen", "symbol": "JP¥", "symbol_native": "¥", "decimal_digits": 0},
			"EUR": {"code": "EUR", "name": "Euro", "symbol": "€", "symbol_native": "€", "decimal_digits": 2}}`))
	})
	mux.HandleFunc("/rates/", func(w http.ResponseWriter, r *http.Request) {
		base := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/rates/"), ".json")
		f.mu.Lock()
		f.hits[base]++
		down, body := f.down, f.files[base]
		_, ok := f.files[base]
		f.mu.Unlock()
		if down {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"date": "2026-01-01", "` + base + `": ` + body + `}`))
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeService) requests(base string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.hits[base]
}

func (f *fakeService) set(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn()
}

// source returns a Source on the fake service with a clock the test moves.
func (f *fakeService) source(clock *time.Time) *Source {
	s := New()
	s.Client = f.srv.Client()
	s.MetaURL, s.RatesURL = f.srv.URL+"/meta.json", f.srv.URL+"/rates/%s.json"
	s.now = func() time.Time { return *clock }
	return s
}

func TestRatesWhatOneUnitWorthBase(t *testing.T) {
	clock := time.Now()
	s := newFakeService(t).source(&clock)
	rates, err := s.Rates(context.Background(), "usd")
	if err != nil {
		t.Fatal(err)
	}
	// The file says 0.8 EUR make a dollar, so a euro is worth 1/0.8 dollars.
	for code, want := range map[string]string{"USD": "1", "EUR": "1.25", "JPY": "0.00625", "BTC": "50000"} {
		if !rates[code].Equal(dec(want)) {
			t.Errorf("1 %s = %s USD, want %s", code, rates[code], want)
		}
	}
	if _, ok := rates["ZERO"]; ok {
		t.Error("a currency with no valid rate must not appear")
	}
}

func TestRatesKeepTwelveSignificantDigits(t *testing.T) {
	clock := time.Now()
	f := newFakeService(t)
	// A currency worth 3e-13 dollars, one worth a third of a dollar.
	f.set(func() { f.files["usd"] = `{"eur": 3, "tiny": 3000000000000}` })
	rates, err := f.source(&clock).Rates(context.Background(), "usd")
	if err != nil {
		t.Fatal(err)
	}
	for code, want := range map[string]string{"EUR": "0.333333333333", "TINY": "0.000000000000333333333333"} {
		if !rates[code].Equal(dec(want)) {
			t.Errorf("%s = %s, want %s", code, rates[code], want)
		}
	}
}

func TestSignificant(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"0.000000003", "0.000000003"},
		{"0.0000000000003", "0.0000000000003"}, // a fixed 12 places would have made this zero
		{"0.333333333333333333", "0.333333333333"},
		{"65000.123456789012345", "65000.1234568"},
		{"0.0067", "0.0067"},
		{"0", "0"},
	} {
		if got := significant(dec(c.in), 12); !got.Equal(dec(c.want)) {
			t.Errorf("significant(%s) = %s, want %s", c.in, got, c.want)
		}
	}
}

// Spaces with the same base share one request; each other base costs one.
func TestSameBaseSharesOneFetch(t *testing.T) {
	clock := time.Now()
	f := newFakeService(t)
	s := f.source(&clock)
	for _, base := range []string{"usd", "jpy", "USD", "usd", "jpy"} {
		if _, err := s.Rates(context.Background(), base); err != nil {
			t.Fatal(err)
		}
	}
	if f.requests("usd") != 1 || f.requests("jpy") != 1 {
		t.Fatalf("requests: usd=%d jpy=%d, want 1 each", f.requests("usd"), f.requests("jpy"))
	}
}

func TestExpireRefetchesEachBase(t *testing.T) {
	clock := time.Now()
	f := newFakeService(t)
	s := f.source(&clock)
	_, _ = s.Rates(context.Background(), "usd")
	f.set(func() { f.files["usd"] = `{"eur": 0.5}` })

	s.Expire()
	for i := 0; i < 3; i++ { // three spaces, one base
		rates, err := s.Rates(context.Background(), "usd")
		if err != nil || !rates["EUR"].Equal(dec("2")) {
			t.Fatalf("EUR = %v, %v after Expire, want 2", rates["EUR"], err)
		}
	}
	if n := f.requests("usd"); n != 2 {
		t.Fatalf("%d requests, want 2 (the first and one after Expire)", n)
	}
}

func TestCacheExpiresAfterTTL(t *testing.T) {
	clock := time.Now()
	f := newFakeService(t)
	s := f.source(&clock)
	_, _ = s.Rates(context.Background(), "usd")
	clock = clock.Add(s.TTL - time.Minute)
	_, _ = s.Rates(context.Background(), "usd")
	if n := f.requests("usd"); n != 1 {
		t.Fatalf("%d requests inside the TTL, want 1", n)
	}
	clock = clock.Add(2 * time.Minute)
	_, _ = s.Rates(context.Background(), "usd")
	if n := f.requests("usd"); n != 2 {
		t.Fatalf("%d requests after the TTL, want 2", n)
	}
}

func TestOldRatesServeWhileServiceDown(t *testing.T) {
	clock := time.Now()
	f := newFakeService(t)
	s := f.source(&clock)
	if _, err := s.Rates(context.Background(), "usd"); err != nil {
		t.Fatal(err)
	}
	f.set(func() { f.down = true })
	s.Expire()
	rates, err := s.Rates(context.Background(), "usd")
	if err != nil || !rates["EUR"].Equal(dec("1.25")) {
		t.Fatalf("stale rates: %v %v", rates["EUR"], err)
	}

	// A failed fetch is not repeated on every call, only after Retry.
	hits := f.requests("usd")
	for i := 0; i < 5; i++ {
		_, _ = s.Rates(context.Background(), "usd")
	}
	if n := f.requests("usd") - hits; n != 0 {
		t.Fatalf("a down service was asked %d more times within Retry", n)
	}
	clock = clock.Add(s.Retry + time.Second)
	_, _ = s.Rates(context.Background(), "usd")
	if n := f.requests("usd") - hits; n != 1 {
		t.Fatalf("%d requests after Retry, want 1", n)
	}
}

func TestRatesUnavailableWithoutAnyData(t *testing.T) {
	clock := time.Now()
	f := newFakeService(t)
	f.set(func() { f.down = true })
	if _, err := f.source(&clock).Rates(context.Background(), "usd"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("got %v", err)
	}
	// The built-in list is never passed off as live rates.
	if _, err := NewOffline().Rates(context.Background(), "usd"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("offline: %v", err)
	}
}

func TestUnknownBaseIsAnError(t *testing.T) {
	clock := time.Now()
	s := newFakeService(t).source(&clock)
	if _, err := s.Rates(context.Background(), "xyz"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("got %v", err)
	}
}

func TestCatalogCombinesNamesLiveRatesUncached(t *testing.T) {
	clock := time.Now()
	f := newFakeService(t)
	s := f.source(&clock)
	list := s.Catalog(context.Background(), "usd")
	if len(list) != 3 {
		t.Fatalf("%d currencies: %+v", len(list), list)
	}
	var jpy bool
	for _, c := range list {
		if c.Code == "JPY" {
			jpy = true
			if c.Name != "Japanese Yen" || c.Symbol != "¥" || c.Decimals != 0 || !c.Rate.Equal(dec("0.00625")) {
				t.Fatalf("JPY: %+v", c)
			}
		}
	}
	if !jpy {
		t.Fatal("no JPY in the catalog")
	}

	// The update cache is separate: a catalog look-up neither fills nor uses it.
	if n := f.requests("usd"); n != 1 {
		t.Fatalf("%d requests for the catalog, want 1", n)
	}
	_, _ = s.Rates(context.Background(), "usd")
	if n := f.requests("usd"); n != 2 {
		t.Fatalf("Rates reused the catalog's fetch: %d requests, want 2", n)
	}
	s.Catalog(context.Background(), "usd")
	if n := f.requests("usd"); n != 3 {
		t.Fatalf("the catalog must look rates up live: %d requests, want 3", n)
	}
}

func TestCatalogFallsBackBuiltInList(t *testing.T) {
	list := NewOffline().Catalog(context.Background(), "eur")
	if len(list) != len(fallback) {
		t.Fatalf("%d currencies, want %d", len(list), len(fallback))
	}
	rate := map[string]decimal.Decimal{}
	for _, c := range list {
		rate[c.Code] = c.Rate
	}
	// In euros: 1 EUR is 1, and 1 USD is worth 0.92 EUR.
	if !rate["EUR"].Equal(dec("1")) || !rate["USD"].Equal(dec("0.92")) {
		t.Fatalf("EUR=%s USD=%s", rate["EUR"], rate["USD"])
	}
}

func TestCatalogStatusMarksFallbackList(t *testing.T) {
	clock := time.Now()
	f := newFakeService(t)
	s := f.source(&clock)

	if _, fallback := s.CatalogStatus(context.Background(), "usd"); fallback {
		t.Fatal("the live list was reported as the built-in one")
	}

	// A server error on the names file: the short built-in list, and it says so.
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(broken.Close)
	down := f.source(&clock)
	down.MetaURL = broken.URL
	list, fallback := down.CatalogStatus(context.Background(), "usd")
	if !fallback || len(list) != len(fallbackMeta()) {
		t.Fatalf("fallback=%v with %d currencies", fallback, len(list))
	}
	// Still the short list, and still said so, until the names can be fetched again.
	if _, again := down.CatalogStatus(context.Background(), "usd"); !again {
		t.Fatal("the flag was lost while the built-in list was in use")
	}
	down.MetaURL = f.srv.URL + "/meta.json"
	clock = clock.Add(down.Retry + time.Second)
	if _, recovered := down.CatalogStatus(context.Background(), "usd"); recovered {
		t.Fatal("still reported as the built-in list after the service came back")
	}
}
