// Package money is the amount type: an exact integer count of a currency's
// minor units together with the currency it is in.
//
// The SQLite schema stores the same thing, minor units scaled by the owning
// currency's decimals. Keeping the unit next to the number means an amount
// cannot be added to another currency's amount, cannot lose its scale and
// cannot be left unrounded.
package money

import (
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/shopspring/decimal"
)

// MaxMinor bounds the amounts users may enter, in minor units of any currency.
// It keeps sums over many rows far from int64 overflow (which SQLite SUM
// reports as an error that would zero a balance) and below 2^53, so the number
// a client receives is still an exact integer for JavaScript.
const MaxMinor int64 = 1_000_000_000_000_000

// ErrOutOfRange is returned for an input amount beyond MaxMinor, and is the
// value of the panic when arithmetic leaves int64.
var ErrOutOfRange = errors.New("amount is out of range")

// RateDivPrecision is the number of fraction digits kept when dividing an
// amount by a currency rate.
const RateDivPrecision = 16

// Unit identifies a currency and its scale: 10^Decimals minor units make one
// major unit. Two amounts are in the same currency exactly when their units
// are equal.
type Unit struct {
	ID       int64
	Decimals int
}

// Money is an amount in a unit. The zero Money has no unit; build amounts
// with New, Zero or FromDecimal.
type Money struct {
	minor int64
	unit  Unit
}

// New is an amount of minor units, as stored.
func New(minor int64, unit Unit) Money { return Money{minor: minor, unit: unit} }

// Zero is no money in unit; start sums from it.
func Zero(unit Unit) Money { return Money{unit: unit} }

// FromDecimal rounds d (major units) half away from zero to the unit's scale.
// It is for values the program computed: one that does not fit int64 panics
// with ErrOutOfRange instead of wrapping around. Use FromInput for user input.
func FromDecimal(d decimal.Decimal, unit Unit) Money {
	scaled := d.Round(int32(unit.Decimals)).Shift(int32(unit.Decimals)).BigInt()
	if !scaled.IsInt64() {
		panic(ErrOutOfRange)
	}
	return Money{minor: scaled.Int64(), unit: unit}
}

// FromInput is FromDecimal for an amount a user entered: beyond MaxMinor in
// either direction it is an ErrOutOfRange error, not a stored value.
func FromInput(d decimal.Decimal, unit Unit) (Money, error) {
	scaled := d.Round(int32(unit.Decimals)).Shift(int32(unit.Decimals)).BigInt()
	if !scaled.IsInt64() || scaled.Int64() > MaxMinor || scaled.Int64() < -MaxMinor {
		return Money{}, ErrOutOfRange
	}
	return Money{minor: scaled.Int64(), unit: unit}, nil
}

// FromNullInput is FromInput for an optional amount; nil stays nil.
func FromNullInput(d *decimal.Decimal, unit Unit) (*Money, error) {
	if d == nil {
		return nil, nil
	}
	m, err := FromInput(*d, unit)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Minor is the stored integer.
func (m Money) Minor() int64 { return m.minor }

// Unit is the currency the amount is in.
func (m Money) Unit() Unit { return m.unit }

// Decimal is the amount in major units, exact. Use it for arithmetic that
// leaves the currency (rates, quantities) and convert back with FromDecimal.
func (m Money) Decimal() decimal.Decimal { return decimal.New(m.minor, -int32(m.unit.Decimals)) }

// Add and Sub panic when the units differ (mixing currencies is a bug in the
// caller, and converting is an explicit step that needs a rate) and with
// ErrOutOfRange when the result leaves int64.
func (m Money) Add(o Money) Money {
	m.mustMatch(o)
	sum := m.minor + o.minor
	if (m.minor^sum)&(o.minor^sum) < 0 {
		panic(ErrOutOfRange)
	}
	m.minor = sum
	return m
}

func (m Money) Sub(o Money) Money {
	m.mustMatch(o)
	diff := m.minor - o.minor
	if (m.minor^o.minor)&(m.minor^diff) < 0 {
		panic(ErrOutOfRange)
	}
	m.minor = diff
	return m
}

func (m Money) Neg() Money {
	if m.minor == math.MinInt64 {
		panic(ErrOutOfRange)
	}
	m.minor = -m.minor
	return m
}

// Abs is the amount without its sign.
func (m Money) Abs() Money {
	if m.minor < 0 {
		return m.Neg()
	}
	return m
}

func (m Money) IsZero() bool     { return m.minor == 0 }
func (m Money) IsPositive() bool { return m.minor > 0 }
func (m Money) IsNegative() bool { return m.minor < 0 }

// Cmp returns -1, 0 or 1; it panics when the units differ.
func (m Money) Cmp(o Money) int {
	m.mustMatch(o)
	switch {
	case m.minor < o.minor:
		return -1
	case m.minor > o.minor:
		return 1
	}
	return 0
}

func (m Money) Equal(o Money) bool { return m.unit == o.unit && m.minor == o.minor }

// Max returns the larger amount.
func Max(a, b Money) Money {
	if a.Cmp(b) >= 0 {
		return a
	}
	return b
}

func (m Money) mustMatch(o Money) {
	if m.unit != o.unit {
		panic(fmt.Sprintf("money: mixed currencies %+v and %+v", m.unit, o.unit))
	}
}

// String is the amount in major units, e.g. "12.5".
func (m Money) String() string { return m.Decimal().String() }

// MarshalJSON writes a JSON number in major units. The number carries no
// currency; the response names it next to the value.
func (m Money) MarshalJSON() ([]byte, error) { return []byte(m.Decimal().String()), nil }

// FromNullMinor reads a nullable stored amount; nil when NULL.
func FromNullMinor(v sql.NullInt64, unit Unit) *Money {
	if !v.Valid {
		return nil
	}
	m := New(v.Int64, unit)
	return &m
}

// ToNullMinor is the stored form of an optional amount.
func ToNullMinor(m *Money) sql.NullInt64 {
	if m == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: m.minor, Valid: true}
}

// FromNullDecimal rounds an optional input amount into unit; nil stays nil.
func FromNullDecimal(d *decimal.Decimal, unit Unit) *Money {
	if d == nil {
		return nil
	}
	m := FromDecimal(*d, unit)
	return &m
}
