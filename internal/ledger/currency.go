package ledger

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/money"
	"savvy-go/internal/store"
)

var one = decimal.NewFromInt(1)

// ErrDecimalsImmutable is returned when an update changes currency decimals.
// Stored minor units are only meaningful together with the decimals, so they
// are fixed when the currency is created.
var ErrDecimalsImmutable = errors.New("currency decimals cannot be changed")

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
		panic(fmt.Sprintf("ledger: converting %+v from currency %+v", m.Unit(), from.Unit()))
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
		panic(fmt.Sprintf("ledger: converting %+v from currency %+v", m.Unit(), from.Unit()))
	}
	if from.ID == to.ID {
		return m, nil
	}
	return money.FromInput(convertedDecimal(m, from, to), to.Unit())
}

func convertedDecimal(m money.Money, from, to Currency) decimal.Decimal {
	return to.ConvertFromBase(from.ConvertToBase(m.Decimal()))
}

// RateSource knows the world's currencies and their exchange rates. It is
// shared by every space of the server (see internal/currencydata).
type RateSource interface {
	// Catalog lists the currencies that exist, each with the rate of one unit
	// in base (zero when unknown). It may be a short built-in list when the
	// real data is unreachable, so it is for choosing currencies, never for
	// updating rates.
	Catalog(ctx context.Context, base string) []Currency
	// Rates returns live rates: for every currency the source has a rate for,
	// how much of base one unit is worth, keyed by upper-case code. It fails
	// when no live rates are available.
	Rates(ctx context.Context, base string) (map[string]decimal.Decimal, error)
}

// Currencies are a space's currencies. Source is only needed to pick
// currencies from the catalog and to update rates.
type Currencies struct {
	DB     store.DB
	Source RateSource
}

func (s Currencies) repo() repo.Repo { return repo.Repo{DB: s.DB} }

func (s Currencies) All(ctx context.Context) ([]Currency, error) {
	rows, err := s.repo().Currencies(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Currency, len(rows))
	for i, r := range rows {
		out[i] = currencyFrom(r)
	}
	return out, nil
}

func (s Currencies) ByID(ctx context.Context, id int64) (*Currency, error) {
	return optCurrency(s.repo().Currency(ctx, id))
}

func (s Currencies) ByCode(ctx context.Context, code string) (*Currency, error) {
	return optCurrency(s.repo().CurrencyByCode(ctx, strings.ToUpper(code)))
}

func (s Currencies) Base(ctx context.Context) (*Currency, error) {
	return optCurrency(s.repo().BaseCurrency(ctx))
}

func optCurrency(r *repo.Currency, err error) (*Currency, error) {
	if r == nil || err != nil {
		return nil, err
	}
	c := currencyFrom(*r)
	return &c, nil
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
	if n, err := s.repo().CountCurrencies(ctx); err == nil && n == 0 {
		c.IsBase = true
	}
	if c.IsBase {
		if err := s.repo().ClearBaseCurrency(ctx); err != nil {
			return nil, err
		}
		c.Rate = one
	}
	id, err := s.repo().InsertCurrency(ctx, currencyRow(c), time.Now().UTC())
	if err != nil {
		return nil, err
	}
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
		if err := s.repo().ClearBaseCurrency(ctx); err != nil {
			return nil, err
		}
		c.Rate = one
	}
	if err := s.repo().UpdateCurrency(ctx, id, currencyRow(c), time.Now().UTC()); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Currencies) Delete(ctx context.Context, id int64) error {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return err
	}
	if s.repo().CurrencyInUse(ctx, id) {
		return fmt.Errorf("in use")
	}
	if cur.IsBase {
		return fmt.Errorf("base")
	}
	return s.repo().DeleteCurrency(ctx, id)
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
	rows, err := s.repo().OtherRates(ctx, id)
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
		if err := s.repo().SetCurrencyRate(ctx, r.ID, rebased[i]); err != nil {
			return nil, err
		}
	}
	if err := s.repo().SetBaseCurrency(ctx, id, time.Now().UTC()); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Currencies) FindOrCreateByCode(ctx context.Context, code string) (*Currency, error) {
	if c, err := s.ByCode(ctx, code); err != nil || c != nil {
		return c, err
	}
	n, _ := s.repo().CountCurrencies(ctx)
	item := s.catalogItem(ctx, code)
	if item == nil {
		return nil, fmt.Errorf("unknown code")
	}
	item.IsBase = n == 0
	if item.IsBase {
		item.Rate = one
	}
	return s.Create(ctx, *item)
}

func (s Currencies) catalogItem(ctx context.Context, code string) *Currency {
	if s.Source == nil {
		return nil
	}
	code = strings.ToUpper(code)
	for _, c := range s.Source.Catalog(ctx, s.baseCode(ctx)) {
		if c.Code == code {
			return &c
		}
	}
	return nil
}

// Catalog lists the currencies of the source the space does not have yet.
func (s Currencies) Catalog(ctx context.Context) []Currency {
	existing := map[string]bool{}
	if codes, err := s.repo().CurrencyCodes(ctx); err == nil {
		for _, code := range codes {
			existing[strings.ToUpper(code)] = true
		}
	}
	out := []Currency{}
	if s.Source == nil {
		return out
	}
	for _, item := range s.Source.Catalog(ctx, s.baseCode(ctx)) {
		if existing[item.Code] {
			continue
		}
		out = append(out, item)
	}
	return out
}

func (s Currencies) baseCode(ctx context.Context) string {
	if b, err := s.Base(ctx); err == nil && b != nil {
		return b.Code
	}
	return "USD"
}

func currencyFrom(r repo.Currency) Currency {
	return Currency{
		ID:       r.ID,
		Code:     r.Code,
		Name:     r.Name,
		Symbol:   r.Symbol,
		Decimals: r.Decimals,
		IsBase:   r.IsBase,
		Rate:     r.Rate,
	}
}

func currencyRow(c Currency) repo.Currency {
	return repo.Currency{
		ID:       c.ID,
		Code:     c.Code,
		Name:     c.Name,
		Symbol:   c.Symbol,
		Decimals: c.Decimals,
		IsBase:   c.IsBase,
		Rate:     c.Rate,
	}
}

// UpdateRates sets the rate of every non-base currency from the source.
// Currencies the source has no valid rate for are skipped and keep theirs.
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
	if s.Source == nil {
		return 0, 0, fmt.Errorf("currency rates unavailable")
	}
	rates, err := s.Source.Rates(ctx, base.Code)
	if err != nil {
		return 0, 0, err
	}
	for _, c := range all {
		if c.ID == base.ID {
			continue
		}
		rate := rates[strings.ToUpper(c.Code)]
		if !validRate(rate) {
			skipped++
			continue
		}
		if err := s.repo().SetCurrencyRate(ctx, c.ID, rate); err != nil {
			return updated, skipped, err
		}
		updated++
	}
	return updated, skipped, nil
}
