// Package currencydata is where the server learns about the world's
// currencies and their exchange rates: names, symbols and decimals from one
// public file, and for each base currency a file with the rate of every other
// currency in it.
//
// There is one Source per server, shared by every space. Rates for updating
// a space are cached in a map by base currency, so spaces with the same base
// share one request; Expire starts a new round. Choosing currencies from the
// catalog looks rates up live and does not touch that cache.
package currencydata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
)

const (
	defaultMetaURL  = "https://cdn.jsdelivr.net/gh/fawazahmed0/exchange-api@main/other/Common-Currency.json"
	defaultRatesURL = "https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest/v1/currencies/%s.json"

	// rateDigits is how many significant digits a stored rate keeps. Rounding
	// to a fixed number of fraction places would turn a small rate (a token
	// worth 3e-13 of the base) into zero.
	rateDigits = 12
	// divPrecision is the fraction digits kept while dividing, far finer than
	// rateDigits so that rounding happens once, at the end.
	divPrecision = 32
)

// ErrUnavailable is returned when no exchange rates can be had: the service is
// unreachable and nothing was fetched before.
var ErrUnavailable = errors.New("currency rates unavailable")

// Source serves currency data. The zero value is not usable; use New.
type Source struct {
	// Client fetches the data.
	Client *http.Client
	// MetaURL is the file of currency names; RatesURL is a format with one %s
	// for the lower-case base code, a file of how many units of every currency
	// make one unit of the base.
	MetaURL, RatesURL string
	// TTL is how long fetched data is kept; Retry is how long to wait after a
	// failed fetch before trying again.
	TTL, Retry time.Duration
	// RefreshEvery is the shortest time between two forced retries (Refresh).
	RefreshEvery time.Duration

	now func() time.Time

	mu        sync.Mutex
	meta      []ledger.Currency
	metaUntil time.Time
	// metaBuiltIn is set while meta is the short built-in list, because the
	// names file has never been fetched.
	metaBuiltIn bool
	// metaFailed is set while the last fetch of the names file failed and its
	// retry window is running.
	metaFailed bool
	// lastRefresh is when Refresh last let a retry through.
	lastRefresh time.Time
	rates       map[string]cachedRates // by upper-case base code; for updates only
}

type cachedRates struct {
	byCode map[string]decimal.Decimal // upper-case code -> value of one unit in the base
	until  time.Time
	failed bool // the last fetch failed; until is the end of the retry window
}

var _ ledger.RateSource = (*Source)(nil)

// New returns a Source for the public data files.
func New() *Source {
	return &Source{
		Client:  &http.Client{Timeout: 10 * time.Second},
		MetaURL: defaultMetaURL, RatesURL: defaultRatesURL,
		TTL: 24 * time.Hour, Retry: 5 * time.Minute, RefreshEvery: 10 * time.Second,
		now:   time.Now,
		rates: map[string]cachedRates{},
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("offline")
}

// NewOffline returns a Source that never touches the network: it serves the
// built-in currency list and reports live rates as unavailable. For tests.
func NewOffline() *Source {
	s := New()
	s.Client = &http.Client{Transport: failingTransport{}}
	return s
}

// Expire marks every cached rate as old, so the next Rates call for each base
// fetches it again, once. The old rates stay in use if that fetch fails. Call
// it at the start of a round of updates.
func (s *Source) Expire() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for base, c := range s.rates {
		c.until = time.Time{}
		s.rates[base] = c
	}
}

// Rates returns, for every currency in base's file, how much of base one unit
// of it is worth (the app's "1 currency = X base"), keyed by upper-case code.
// They are cached by base: spaces with the same base share one request.
func (s *Source) Rates(ctx context.Context, base string) (map[string]decimal.Decimal, error) {
	base = strings.ToUpper(strings.TrimSpace(base))
	s.mu.Lock()
	defer s.mu.Unlock()
	c, cached := s.rates[base]
	if cached && s.now().Before(c.until) {
		return c.byCode, nil
	}
	byCode, err := s.fetchRates(ctx, base)
	if err != nil {
		// Keep what we have, and do not ask again until Retry has passed.
		c.until, c.failed = s.now().Add(s.Retry), true
		s.rates[base] = c
		if c.byCode != nil {
			return c.byCode, nil
		}
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	s.rates[base] = cachedRates{byCode: byCode, until: s.now().Add(s.TTL)}
	return byCode, nil
}

// Catalog lists every currency known, with its rate in base (zero where no
// rate is known). The rates are looked up live, not cached. When the service
// is unreachable it falls back to a short built-in list.
func (s *Source) Catalog(ctx context.Context, base string) []ledger.Currency {
	list, _ := s.CatalogStatus(ctx, base)
	return list
}

// CatalogStatus is Catalog, and says whether the list is the short built-in
// one because the names could not be fetched, so the caller can tell the user
// that a currency may be missing.
func (s *Source) CatalogStatus(ctx context.Context, base string) (list []ledger.Currency, builtIn bool) {
	base = strings.ToUpper(strings.TrimSpace(base))
	s.mu.Lock()
	defer s.mu.Unlock()
	meta := s.metaList(ctx)
	rates, err := s.fetchRates(ctx, base)
	if err != nil {
		rates, _ = rebase(fallbackPerUSD(), base)
	}
	out := make([]ledger.Currency, len(meta))
	for i, c := range meta {
		c.Rate = rates[c.Code]
		out[i] = c
	}
	return out, s.metaBuiltIn
}

// fetchRates loads base's file: its numbers say how many units of each
// currency make one base, so one unit of a currency is worth 1/number base.
func (s *Source) fetchRates(ctx context.Context, base string) (map[string]decimal.Decimal, error) {
	key := strings.ToLower(base)
	var doc map[string]json.RawMessage
	if err := s.fetchJSON(ctx, fmt.Sprintf(s.RatesURL, key), &doc); err != nil {
		return nil, err
	}
	var units map[string]json.Number
	if err := json.Unmarshal(doc[key], &units); err != nil {
		return nil, fmt.Errorf("currency data: no rates for %s", base)
	}
	out := make(map[string]decimal.Decimal, len(units)+1)
	for code, raw := range units {
		v, err := decimal.NewFromString(raw.String())
		if err != nil || !v.IsPositive() {
			continue
		}
		out[strings.ToUpper(code)] = significant(decimal.NewFromInt(1).DivRound(v, divPrecision), rateDigits)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("currency data: no rates for %s", base)
	}
	out[base] = decimal.NewFromInt(1)
	return out, nil
}

type remoteCurrency struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Symbol        string `json:"symbol"`
	SymbolNative  string `json:"symbol_native"`
	DecimalDigits int    `json:"decimal_digits"`
}

// metaList returns the currencies' names, symbols and decimals. The caller
// holds the lock.
func (s *Source) metaList(ctx context.Context) []ledger.Currency {
	if s.now().Before(s.metaUntil) {
		return s.meta
	}
	var raw map[string]remoteCurrency
	if err := s.fetchJSON(ctx, s.MetaURL, &raw); err != nil || len(raw) == 0 {
		if s.meta == nil {
			s.meta = fallbackMeta()
			s.metaBuiltIn = true
		}
		s.metaUntil, s.metaFailed = s.now().Add(s.Retry), true
		return s.meta
	}
	list := make([]ledger.Currency, 0, len(raw))
	for key, r := range raw {
		code := strings.ToUpper(strings.TrimSpace(r.Code))
		if code == "" {
			code = strings.ToUpper(key)
		}
		symbol := r.SymbolNative
		if symbol == "" {
			symbol = r.Symbol
		}
		list = append(list, ledger.Currency{Code: code, Name: r.Name, Symbol: symbol, Decimals: r.DecimalDigits})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Code < list[j].Code })
	s.meta = list
	s.metaBuiltIn, s.metaFailed = false, false
	s.metaUntil = s.now().Add(s.TTL)
	return list
}

func (s *Source) fetchJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	res, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("currency data: %s", res.Status)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

// rebase turns the built-in units-per-dollar list into the value of one unit
// of each currency in base: with A the units per dollar, one unit of code is
// A[base]/A[code] base units. ok is false when base is not in the list. Only
// the built-in fallback needs this; live rates come per base from the service.
func rebase(perUSD map[string]decimal.Decimal, base string) (map[string]decimal.Decimal, bool) {
	baseUnits, ok := perUSD[strings.ToUpper(base)]
	if !ok {
		return nil, false
	}
	out := make(map[string]decimal.Decimal, len(perUSD))
	for code, units := range perUSD {
		out[code] = significant(baseUnits.DivRound(units, divPrecision), rateDigits)
	}
	return out, true
}

// significant rounds d to n significant digits.
func significant(d decimal.Decimal, n int) decimal.Decimal {
	if d.IsZero() {
		return d
	}
	magnitude := len(d.Abs().Coefficient().String()) + int(d.Exponent()) // digits before the point
	return d.Round(int32(n - magnitude))
}

// Refresh ends the wait after a failed fetch, so the next call for that data
// tries the network again, once. It leaves alone everything that was fetched
// fine (that data keeps its TTL), and old data still serves if the retry fails.
// To keep the upstream service from being hammered, it lets one retry through
// per RefreshEvery; inside that window it changes nothing and returns false.
func (s *Source) Refresh() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if !s.lastRefresh.IsZero() && now.Before(s.lastRefresh.Add(s.RefreshEvery)) {
		return false
	}
	s.lastRefresh = now
	if s.metaFailed {
		s.metaUntil = time.Time{}
	}
	for base, c := range s.rates {
		if c.failed {
			c.until = time.Time{}
			s.rates[base] = c
		}
	}
	return true
}
