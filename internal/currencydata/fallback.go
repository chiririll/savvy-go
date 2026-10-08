package currencydata

import (
	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
)

// fallback is the built-in list, used for choosing currencies when the
// service is unreachable. Rate is units of the currency per 1 US dollar. It
// is never used to update the rates a space already has.
var fallback = []struct {
	Code, Name, Symbol string
	Decimals           int
	PerUSD             string
}{
	{"USD", "US Dollar", "$", 2, "1"},
	{"EUR", "Euro", "€", 2, "0.92"},
	{"GBP", "British Pound", "£", 2, "0.79"},
	{"UAH", "Ukrainian Hryvnia", "₴", 2, "41"},
	{"PLN", "Polish Zloty", "zł", 2, "4"},
	{"JPY", "Japanese Yen", "¥", 0, "150"},
	{"CHF", "Swiss Franc", "CHF", 2, "0.88"},
	{"RUB", "Russian Ruble", "₽", 2, "84.51"},
	{"BTC", "Bitcoin", "₿", 8, "0.000015"},
}

func fallbackMeta() []ledger.Currency {
	out := make([]ledger.Currency, len(fallback))
	for i, c := range fallback {
		out[i] = ledger.Currency{Code: c.Code, Name: c.Name, Symbol: c.Symbol, Decimals: c.Decimals}
	}
	return out
}

func fallbackPerUSD() map[string]decimal.Decimal {
	out := make(map[string]decimal.Decimal, len(fallback))
	for _, c := range fallback {
		out[c.Code] = decimal.RequireFromString(c.PerUSD)
	}
	return out
}
