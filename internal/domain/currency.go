package domain

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/money"
	"savvy-go/internal/store"
)

var one = decimal.NewFromInt(1)

// ErrDecimalsImmutable is returned when an update changes currency decimals.
// Stored minor units are only meaningful together with the decimals, so they
// are fixed when the currency is created.
var ErrDecimalsImmutable = errors.New("currency decimals cannot be changed")

// rateFromFloat converts an externally supplied (API) float rate to a decimal
// of 12 significant digits. Rounding to a fixed number of fraction places would
// turn a small rate (a token worth 3e-13 of the base) into zero.
func rateFromFloat(v float64) decimal.Decimal {
	d, err := decimal.NewFromString(strconv.FormatFloat(v, 'g', 12, 64))
	if err != nil {
		return decimal.Zero
	}
	return d
}

const (
	// MaxDecimals is how many fraction digits a currency may have: the schema
	// allows 0 to 12, which keeps MaxMinor amounts within int64.
	MaxDecimals = 12
	// rebasePrecision is the fraction digits kept when rates are re-expressed
	// against a new base currency. It is far finer than money.RateDivPrecision:
	// a rate of 1e-13 still needs several significant digits.
	rebasePrecision = 32
)

// Rates outside this range cannot be told from zero or overflow an amount
// converted at them.
var (
	minRate = decimal.New(1, -15)
	maxRate = decimal.New(1, 15)
)

var (
	// ErrInvalidDecimals is returned for a currency with decimals out of range.
	ErrInvalidDecimals = errors.New("currency decimals must be between 0 and 12")
	// ErrInvalidRate is returned for a rate that is not within [1e-15, 1e15].
	ErrInvalidRate = errors.New("currency rate must be between 0.000000000000001 and 1000000000000000")
)

func validRate(r decimal.Decimal) bool { return !r.LessThan(minRate) && !r.GreaterThan(maxRate) }

type Currency struct {
	ID       int64
	Code     string
	Name     string
	Symbol   string
	Decimals int
	IsBase   bool
	Rate     decimal.Decimal
}

func (c Currency) ConvertToBase(amount decimal.Decimal) decimal.Decimal {
	if c.IsBase {
		return amount
	}
	return amount.Mul(c.Rate)
}

func (c Currency) ConvertFromBase(amount decimal.Decimal) decimal.Decimal {
	if c.IsBase || c.Rate.IsZero() {
		return amount
	}
	return amount.DivRound(c.Rate, money.RateDivPrecision)
}

// Unit is the scale amounts in this currency are stored in.
func (c Currency) Unit() money.Unit { return money.Unit{ID: c.ID, Decimals: c.Decimals} }

// Convert re-expresses m, an amount in from, in to, rounded to the scale of to.
// It panics with money.ErrOutOfRange when the result does not fit; TryConvert
// reports that as an error, for amounts a user chose.
func Convert(m money.Money, from, to Currency) money.Money {
	if m.Unit() != from.Unit() {
		panic(fmt.Sprintf("domain: converting %+v from currency %+v", m.Unit(), from.Unit()))
	}
	if from.ID == to.ID {
		return m
	}
	return money.FromDecimal(convertedDecimal(m, from, to), to.Unit())
}

// TryConvert is Convert with the range as an error: money.ErrOutOfRange when
// the result is beyond money.MaxMinor.
func TryConvert(m money.Money, from, to Currency) (money.Money, error) {
	if m.Unit() != from.Unit() {
		panic(fmt.Sprintf("domain: converting %+v from currency %+v", m.Unit(), from.Unit()))
	}
	if from.ID == to.ID {
		return m, nil
	}
	return money.FromInput(convertedDecimal(m, from, to), to.Unit())
}

func convertedDecimal(m money.Money, from, to Currency) decimal.Decimal {
	return to.ConvertFromBase(from.ConvertToBase(m.Decimal()))
}

type Currencies struct{ DB store.DB }

func (s Currencies) All(ctx context.Context) ([]Currency, error) {
	rows, err := db.Q(s.DB).ListCurrencies(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Currency, 0, len(rows))
	for _, r := range rows {
		out = append(out, currencyFrom(r.ID, r.Code, r.Name, r.Symbol, r.Decimals, r.IsBase, r.Rate))
	}
	return out, nil
}

func (s Currencies) ByID(ctx context.Context, id int64) (*Currency, error) {
	r, err := db.Q(s.DB).GetCurrency(ctx, id)
	return currencyFromRow(r.ID, r.Code, r.Name, r.Symbol, r.Decimals, r.IsBase, r.Rate, err)
}

func (s Currencies) ByCode(ctx context.Context, code string) (*Currency, error) {
	r, err := db.Q(s.DB).GetCurrencyByCode(ctx, strings.ToUpper(code))
	return currencyFromRow(r.ID, r.Code, r.Name, r.Symbol, r.Decimals, r.IsBase, r.Rate, err)
}

func (s Currencies) Base(ctx context.Context) (*Currency, error) {
	r, err := db.Q(s.DB).GetBaseCurrency(ctx)
	return currencyFromRow(r.ID, r.Code, r.Name, r.Symbol, r.Decimals, r.IsBase, r.Rate, err)
}

func (s Currencies) Create(ctx context.Context, c Currency) (*Currency, error) {
	c.Code = strings.ToUpper(strings.TrimSpace(c.Code))
	if c.Decimals < 0 {
		c.Decimals = 2
	}
	if c.Decimals > MaxDecimals {
		return nil, ErrInvalidDecimals
	}
	if c.Rate.IsZero() {
		c.Rate = one
	}
	if !validRate(c.Rate) {
		return nil, ErrInvalidRate
	}
	if n, err := db.Q(s.DB).CountCurrencies(ctx); err == nil && n == 0 {
		c.IsBase = true
	}
	if c.IsBase {
		if err := db.Q(s.DB).ClearBaseCurrency(ctx); err != nil {
			return nil, err
		}
		c.Rate = one
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Q(s.DB).InsertCurrency(ctx, sqlc.InsertCurrencyParams{
		Code: c.Code, Name: c.Name, Symbol: c.Symbol, Decimals: int64(c.Decimals),
		IsBase: db.BoolInt(c.IsBase), Rate: c.Rate, CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
	})
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

func (s Currencies) Update(ctx context.Context, id int64, c Currency) (*Currency, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if c.Decimals != cur.Decimals {
		return nil, ErrDecimalsImmutable
	}
	if !c.IsBase && !cur.IsBase && !validRate(c.Rate) {
		return nil, ErrInvalidRate
	}
	if cur.IsBase && !c.IsBase {
		return nil, fmt.Errorf("cannot unset base")
	}
	if cur.IsBase && !c.Rate.Equal(one) {
		return nil, fmt.Errorf("base rate")
	}
	if c.IsBase && !cur.IsBase {
		if err := db.Q(s.DB).ClearBaseCurrency(ctx); err != nil {
			return nil, err
		}
		c.Rate = one
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err = db.Q(s.DB).UpdateCurrency(ctx, sqlc.UpdateCurrencyParams{
		Name: c.Name, Symbol: c.Symbol, Decimals: int64(c.Decimals), IsBase: db.BoolInt(c.IsBase),
		Rate: c.Rate, UpdatedAt: db.NS(now), ID: id,
	})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Currencies) Delete(ctx context.Context, id int64) error {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return err
	}
	used, _ := db.Q(s.DB).CountAccountsForCurrency(ctx, id)
	usedByBudgets, _ := db.Q(s.DB).CountBudgetsForCurrency(ctx, id)
	if used > 0 || usedByBudgets > 0 {
		return fmt.Errorf("in use")
	}
	if cur.IsBase {
		return fmt.Errorf("base")
	}
	return db.Q(s.DB).DeleteCurrency(ctx, id)
}

func (s Currencies) SetBase(ctx context.Context, id int64) (*Currency, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if cur.IsBase {
		return cur, nil
	}
	newRate := cur.Rate
	if newRate.IsZero() {
		newRate = one
	}
	rows, err := db.Q(s.DB).ListOtherCurrencyRates(ctx, id)
	if err != nil {
		return nil, err
	}
	rebased := make([]decimal.Decimal, len(rows))
	for i, r := range rows {
		rebased[i] = r.Rate.DivRound(newRate, rebasePrecision)
		if !validRate(rebased[i]) {
			return nil, ErrInvalidRate // nothing is written: rates stay as they were
		}
	}
	for i, r := range rows {
		if err := db.Q(s.DB).UpdateCurrencyRate(ctx, sqlc.UpdateCurrencyRateParams{Rate: rebased[i], ID: r.ID}); err != nil {
			return nil, err
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err = db.Q(s.DB).SetCurrencyBase(ctx, sqlc.SetCurrencyBaseParams{UpdatedAt: db.NS(now), ID: id})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Currencies) FindOrCreateByCode(ctx context.Context, code string) (*Currency, error) {
	if c, err := s.ByCode(ctx, code); err != nil || c != nil {
		return c, err
	}
	n, _ := db.Q(s.DB).CountCurrencies(ctx)
	item := catalogItem(ctx, s.baseCode(ctx), code)
	if item == nil {
		return nil, fmt.Errorf("unknown code")
	}
	item.IsBase = n == 0
	if item.IsBase {
		item.Rate = one
	}
	return s.Create(ctx, *item)
}

func (s Currencies) Catalog(ctx context.Context) []map[string]any {
	existing := map[string]bool{}
	if rows, err := db.Q(s.DB).ListCurrencyCodes(ctx); err == nil {
		for _, code := range rows {
			existing[strings.ToUpper(code)] = true
		}
	}
	out := []map[string]any{}
	for _, item := range loadCatalog(ctx, s.baseCode(ctx)) {
		if existing[item.Code] {
			continue
		}
		out = append(out, map[string]any{
			"code": item.Code, "name": item.Name, "symbol": item.Symbol,
			"decimals": item.Decimals, "rate": item.Rate,
		})
	}
	return out
}

func (s Currencies) baseCode(ctx context.Context) string {
	if b, err := s.Base(ctx); err == nil && b != nil {
		return b.Code
	}
	return "usd"
}

func currencyFrom(id int64, code, name, symbol string, decimals, isBase int64, rate decimal.Decimal) Currency {
	return Currency{ID: id, Code: code, Name: name, Symbol: symbol, Decimals: int(decimals), IsBase: isBase != 0, Rate: rate}
}

func currencyFromRow(id int64, code, name, symbol string, decimals, isBase int64, rate decimal.Decimal, err error) (*Currency, error) {
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c := currencyFrom(id, code, name, symbol, decimals, isBase, rate)
	return &c, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// UpdateRates pulls fresh rates for every non-base currency from the exchange
// API. Currencies the API does not know are left untouched.
func (s Currencies) UpdateRates(ctx context.Context) (updated, skipped int, err error) {
	base, err := s.Base(ctx)
	if err != nil {
		return 0, 0, err
	}
	if base == nil {
		return 0, 0, fmt.Errorf("no base currency")
	}
	all, err := s.All(ctx)
	if err != nil {
		return 0, 0, err
	}
	rates := refreshedRates(ctx, base.Code, all)
	if rates == nil {
		return 0, 0, fmt.Errorf("currency rates unavailable")
	}
	for _, c := range all {
		if c.ID == base.ID {
			continue
		}
		rate := rateFromFloat(rates[strings.ToLower(c.Code)])
		if !validRate(rate) {
			skipped++
			continue
		}
		if err := db.Q(s.DB).UpdateCurrencyRate(ctx, sqlc.UpdateCurrencyRateParams{Rate: rate, ID: c.ID}); err != nil {
			return updated, skipped, err
		}
		updated++
	}
	return updated, skipped, nil
}
