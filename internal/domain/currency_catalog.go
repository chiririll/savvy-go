package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	currencyMetaURL  = "https://cdn.jsdelivr.net/gh/fawazahmed0/exchange-api@main/other/Common-Currency.json"
	currencyRatesURL = "https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest/v1/currencies/%s.json"
	catalogTTL       = 24 * time.Hour
	catalogRetry     = 5 * time.Minute
)

var catalogHTTP = &http.Client{Timeout: 10 * time.Second}

// fallbackCatalog is used only when the remote currency data is unreachable.
// Rate here is units of the currency per 1 USD; loadCatalog rebases it.
var fallbackCatalog = []Currency{
	{Code: "USD", Name: "US Dollar", Symbol: "$", Decimals: 2, Rate: 1},
	{Code: "EUR", Name: "Euro", Symbol: "€", Decimals: 2, Rate: 0.92},
	{Code: "GBP", Name: "British Pound", Symbol: "£", Decimals: 2, Rate: 0.79},
	{Code: "UAH", Name: "Ukrainian Hryvnia", Symbol: "₴", Decimals: 2, Rate: 41},
	{Code: "PLN", Name: "Polish Zloty", Symbol: "zł", Decimals: 2, Rate: 4},
	{Code: "JPY", Name: "Japanese Yen", Symbol: "¥", Decimals: 0, Rate: 150},
	{Code: "CHF", Name: "Swiss Franc", Symbol: "CHF", Decimals: 2, Rate: 0.88},
	{Code: "RUB", Name: "Russian Ruble", Symbol: "₽", Decimals: 2, Rate: 84.51},
	{Code: "BTC", Name: "Bitcoin", Symbol: "₿", Decimals: 8, Rate: 0.000015},
}

type remoteCurrency struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Symbol        string `json:"symbol"`
	SymbolNative  string `json:"symbol_native"`
	DecimalDigits int    `json:"decimal_digits"`
}

var catalogCache struct {
	sync.Mutex
	meta      []Currency
	metaUntil time.Time
	rates     map[string]map[string]float64
	ratesAt   map[string]time.Time
}

func fetchJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	res, err := catalogHTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("currency data: %s", res.Status)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

// catalogMeta returns the cached currency list, refreshing it when stale.
// Must be called with catalogCache locked.
func catalogMeta(ctx context.Context) []Currency {
	if time.Now().Before(catalogCache.metaUntil) {
		return catalogCache.meta
	}
	var raw map[string]remoteCurrency
	if err := fetchJSON(ctx, currencyMetaURL, &raw); err != nil || len(raw) == 0 {
		if catalogCache.meta == nil {
			catalogCache.meta = fallbackCatalog
		}
		catalogCache.metaUntil = time.Now().Add(catalogRetry)
		return catalogCache.meta
	}
	list := make([]Currency, 0, len(raw))
	for key, r := range raw {
		code := strings.ToUpper(strings.TrimSpace(r.Code))
		if code == "" {
			code = strings.ToUpper(key)
		}
		symbol := r.SymbolNative
		if symbol == "" {
			symbol = r.Symbol
		}
		list = append(list, Currency{Code: code, Name: r.Name, Symbol: symbol, Decimals: r.DecimalDigits})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Code < list[j].Code })
	catalogCache.meta = list
	catalogCache.metaUntil = time.Now().Add(catalogTTL)
	return list
}

// catalogRates returns, for every currency, how much of base one unit of it is
// worth (the app's "1 currency = X base"), keyed by lower-case code. The API
// returns the inverse, so values are flipped. Must be called with catalogCache
// locked. force bypasses the cache.
func catalogRates(ctx context.Context, base string, force bool) map[string]float64 {
	base = strings.ToLower(base)
	if base == "" {
		return nil
	}
	if catalogCache.rates == nil {
		catalogCache.rates = map[string]map[string]float64{}
		catalogCache.ratesAt = map[string]time.Time{}
	}
	if at, ok := catalogCache.ratesAt[base]; ok && !force && time.Now().Before(at) {
		return catalogCache.rates[base]
	}
	var raw map[string]json.RawMessage
	if err := fetchJSON(ctx, fmt.Sprintf(currencyRatesURL, base), &raw); err == nil {
		perBase := map[string]float64{}
		_ = json.Unmarshal(raw[base], &perBase)
		rates := make(map[string]float64, len(perBase))
		for code, v := range perBase {
			if v > 0 {
				rates[code] = 1 / v
			}
		}
		if len(rates) > 0 {
			rates[base] = 1
			catalogCache.rates[base] = rates
			catalogCache.ratesAt[base] = time.Now().Add(catalogTTL)
			return rates
		}
	}
	catalogCache.ratesAt[base] = time.Now().Add(catalogRetry)
	return catalogCache.rates[base]
}

// fallbackRates rebases the static fallback list onto baseCode; nil if the
// base is not in it.
func fallbackRates(baseCode string) map[string]float64 {
	var baseUnits float64
	for _, c := range fallbackCatalog {
		if c.Code == strings.ToUpper(baseCode) {
			baseUnits = c.Rate
		}
	}
	if baseUnits == 0 {
		return nil
	}
	out := make(map[string]float64, len(fallbackCatalog))
	for _, c := range fallbackCatalog {
		out[strings.ToLower(c.Code)] = baseUnits / c.Rate
	}
	return out
}

// loadCatalog returns the currency catalog with rates expressed in baseCode
// (1 currency = Rate base). Rate is 0 for currencies without a known rate.
func loadCatalog(ctx context.Context, baseCode string) []Currency {
	catalogCache.Lock()
	defer catalogCache.Unlock()
	meta := catalogMeta(ctx)
	rates := catalogRates(ctx, baseCode, false)
	if rates == nil {
		rates = fallbackRates(baseCode)
	}
	out := make([]Currency, len(meta))
	for i, c := range meta {
		c.Rate = rates[strings.ToLower(c.Code)]
		out[i] = c
	}
	return out
}

// refreshedRates fetches fresh rates against baseCode, bypassing the cache. If
// the API has no data for the base itself, it derives them through a
// reference currency that is present in the app.
func refreshedRates(ctx context.Context, baseCode string, present []Currency) map[string]float64 {
	catalogCache.Lock()
	defer catalogCache.Unlock()
	if rates := catalogRates(ctx, baseCode, true); rates != nil {
		return rates
	}
	base := strings.ToLower(baseCode)
	for _, ref := range []string{"usd", "eur", "gbp", "jpy", "cny"} {
		have := false
		for _, c := range present {
			if strings.EqualFold(c.Code, ref) {
				have = true
			}
		}
		if !have {
			continue
		}
		refRates := catalogRates(ctx, ref, true) // 1 ref-unit code = refRates[code] ref
		baseInRef, ok := refRates[base]
		if !ok || baseInRef == 0 {
			continue
		}
		out := make(map[string]float64, len(refRates))
		for code, v := range refRates {
			out[code] = v / baseInRef
		}
		out[base] = 1
		return out
	}
	return nil
}

func catalogItem(ctx context.Context, baseCode, code string) *Currency {
	code = strings.ToUpper(code)
	for _, c := range loadCatalog(ctx, baseCode) {
		if c.Code == code {
			cp := c
			return &cp
		}
	}
	return nil
}
