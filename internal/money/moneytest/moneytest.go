// Package moneytest generates the currencies and amounts money tests run over.
//
// A money bug hides in the corners: a currency with no decimals, one with many,
// a rate that is tiny or has a long fraction, an amount of one minor unit, one
// that rounds up, one at the allowed maximum. Instead of each test picking its
// own USD-and-a-couple, tests iterate Currencies, Amounts and pairs of them,
// and every case shows up as a named subtest.
//
// The package knows nothing about the application's storage, so tests of any
// package (including domain itself) can import it.
package moneytest

import (
	"fmt"
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/money"
)

// Currency is a currency to run a test against. Rate is the value of one unit
// in USD; use RateIn to express it against another base.
type Currency struct {
	Code     string
	Decimals int
	Rate     decimal.Decimal
	// Note says what corner the currency covers.
	Note string
}

// Unit is the scale of the currency; id is whatever id the test stored it under.
func (c Currency) Unit(id int64) money.Unit { return money.Unit{ID: id, Decimals: c.Decimals} }

// RateIn is the value of one unit of c in base units, as stored for a database
// whose base currency is base. It keeps 32 fraction digits: a rate of 1e-13
// would be left with three significant digits at 16.
func (c Currency) RateIn(base Currency) decimal.Decimal {
	return c.Rate.DivRound(base.Rate, 32)
}

func (c Currency) String() string { return fmt.Sprintf("%s(%d)", c.Code, c.Decimals) }

func rate(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// Currencies is the matrix. It spans the scales the schema allows (0 to 12
// decimals) and rates from a hundred-thousandth to sixty-five thousand.
var Currencies = []Currency{
	{"USD", 2, rate("1"), "the common case, base of the rates"},
	{"JPY", 0, rate("0.0067"), "no decimals, rate with a long fraction"},
	{"KWD", 3, rate("3.25"), "three decimals, strong currency"},
	{"CLF", 4, rate("38.7"), "four decimals"},
	{"BTC", 8, rate("65000"), "eight decimals, very large rate"},
	{"XTW", 12, rate("0.000000003"), "twelve decimals (the maximum), tiny rate"},
	{"IRR", 0, rate("0.0000238"), "no decimals, huge nominal amounts per dollar"},
}

// Amount is one input amount and what it must become.
type Amount struct {
	Name string
	// Input is the amount as a user enters it, in major units; it may carry
	// more digits than the currency's scale.
	Input decimal.Decimal
	// Minor is what Input rounds to, half away from zero, in minor units.
	// Meaningful only when Valid.
	Minor int64
	// Valid is false for an amount beyond money.MaxMinor, which must be
	// rejected rather than stored.
	Valid bool
}

// Negated is the same amount with the opposite sign.
func (a Amount) Negated() Amount {
	a.Name += " (negative)"
	a.Input = a.Input.Neg()
	a.Minor = -a.Minor
	return a
}

func exact(name string, minor int64, decimals int) Amount {
	return Amount{Name: name, Input: decimal.New(minor, -int32(decimals)), Minor: minor, Valid: true}
}

// Amounts are the positive corner cases of a currency with the given decimals:
// the smallest unit, amounts that round down, up and at the tie, a whole unit,
// a typical sum, repeating digits, a large one, the maximum, and two beyond it.
func Amounts(decimals int) []Amount {
	// n tenths of a minor unit, as a decimal of the scale one digit finer.
	tenths := func(name string, n int64, minor int64) Amount {
		return Amount{Name: name, Input: decimal.New(n, -int32(decimals+1)), Minor: minor, Valid: true}
	}
	whole := int64(1)
	for i := 0; i < decimals; i++ {
		whole *= 10
	}
	return []Amount{
		tenths("below half a unit rounds to zero", 4, 0),
		exact("smallest unit", 1, decimals),
		tenths("half a unit rounds up", 5, 1),
		tenths("above half a unit rounds up", 6, 1),
		tenths("one and a half units rounds to two", 15, 2),
		exact("one major unit", whole, decimals),
		exact("typical", 12345, decimals),
		exact("repeating digits", 3333333, decimals),
		exact("large", 987_654_321_012, decimals),
		exact("maximum", money.MaxMinor, decimals),
		{Name: "one unit over the maximum", Input: decimal.New(money.MaxMinor+1, -int32(decimals))},
		{Name: "beyond int64", Input: decimal.RequireFromString("1e30")},
	}
}

// Valid are the amounts a currency must accept.
func Valid(decimals int) []Amount {
	var out []Amount
	for _, a := range Amounts(decimals) {
		if a.Valid {
			out = append(out, a)
		}
	}
	return out
}

// Invalid are the amounts a currency must refuse.
func Invalid(decimals int) []Amount {
	var out []Amount
	for _, a := range Amounts(decimals) {
		if !a.Valid {
			out = append(out, a)
		}
	}
	return out
}

// ForEach runs fn as a subtest for every currency.
func ForEach(t *testing.T, fn func(t *testing.T, c Currency)) {
	t.Helper()
	for _, c := range Currencies {
		t.Run(c.Code, func(t *testing.T) { fn(t, c) })
	}
}

// ForEachAmount runs fn as a subtest for every amount in amounts.
func ForEachAmount(t *testing.T, amounts []Amount, fn func(t *testing.T, a Amount)) {
	t.Helper()
	for _, a := range amounts {
		t.Run(a.Name, func(t *testing.T) { fn(t, a) })
	}
}

// ForEachPair runs fn as a subtest for every ordered pair of currencies,
// including a currency with itself.
func ForEachPair(t *testing.T, fn func(t *testing.T, from, to Currency)) {
	t.Helper()
	for _, from := range Currencies {
		for _, to := range Currencies {
			t.Run(from.Code+"->"+to.Code, func(t *testing.T) { fn(t, from, to) })
		}
	}
}

// Converted is amount (major units of from) in major units of to at the rates
// of the two currencies, unrounded; round it with money.FromDecimal to get the
// amount a conversion must produce.
func Converted(amount decimal.Decimal, from, to Currency) decimal.Decimal {
	return amount.Mul(from.Rate).DivRound(to.Rate, 40)
}

// Fits reports whether d, rounded to decimals, is within money.MaxMinor.
func Fits(d decimal.Decimal, decimals int) bool {
	_, err := money.FromInput(d, money.Unit{Decimals: decimals})
	return err == nil
}
