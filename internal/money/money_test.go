package money_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/money"
	"savvy-go/internal/money/moneytest"
)

var (
	usd = money.Unit{ID: 1, Decimals: 2}
	jpy = money.Unit{ID: 2, Decimals: 0}
	btc = money.Unit{ID: 3, Decimals: 8}
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func mustPanicWith(t *testing.T, want error, fn func()) {
	t.Helper()
	defer func() {
		got, _ := recover().(error)
		if !errors.Is(got, want) {
			t.Fatalf("panic = %v, want %v", got, want)
		}
	}()
	fn()
}

func TestFromDecimalRoundsPerCurrencyAndAmount(t *testing.T) {
	moneytest.ForEach(t, func(t *testing.T, c moneytest.Currency) {
		unit := c.Unit(1)
		moneytest.ForEachAmount(t, moneytest.Valid(c.Decimals), func(t *testing.T, a moneytest.Amount) {
			for _, a := range []moneytest.Amount{a, a.Negated()} {
				m := money.FromDecimal(a.Input, unit)
				if m.Minor() != a.Minor {
					t.Fatalf("%s: minor = %d, want %d", a.Name, m.Minor(), a.Minor)
				}
				if m.Unit() != unit {
					t.Fatalf("unit = %+v, want %+v", m.Unit(), unit)
				}
				// Reading it back is exact: the major-unit value has exactly
				// Decimals fraction digits at most.
				if !m.Decimal().Equal(decimal.New(a.Minor, -int32(c.Decimals))) {
					t.Fatalf("Decimal() = %s", m.Decimal())
				}
			}
		})
	})
}

func TestFromInputRejectsAmountsOutOfRange(t *testing.T) {
	moneytest.ForEach(t, func(t *testing.T, c moneytest.Currency) {
		unit := c.Unit(1)
		moneytest.ForEachAmount(t, moneytest.Amounts(c.Decimals), func(t *testing.T, a moneytest.Amount) {
			for _, a := range []moneytest.Amount{a, a.Negated()} {
				m, err := money.FromInput(a.Input, unit)
				if !a.Valid {
					if !errors.Is(err, money.ErrOutOfRange) {
						t.Fatalf("%s: err = %v, want ErrOutOfRange", a.Name, err)
					}
					continue
				}
				if err != nil || m.Minor() != a.Minor {
					t.Fatalf("%s: got %d, %v; want %d", a.Name, m.Minor(), err, a.Minor)
				}
			}
		})
	})
}

func TestFromDecimalPanicsInsteadOfWrapping(t *testing.T) {
	// These wrapped around silently (to a negative or a wrong amount).
	for _, s := range []string{"92233720368547758.08", "1e30", "-92233720368547758.09"} {
		mustPanicWith(t, money.ErrOutOfRange, func() { money.FromDecimal(dec(s), usd) })
	}
	mustPanicWith(t, money.ErrOutOfRange, func() { money.FromDecimal(dec("1e30"), jpy) })
	// The edges that fit are fine.
	if m := money.FromDecimal(dec("92233720368547758.07"), usd); m.Minor() != math.MaxInt64 {
		t.Errorf("max int64 = %d", m.Minor())
	}
	if m := money.FromDecimal(dec("-92233720368547758.08"), usd); m.Minor() != math.MinInt64 {
		t.Errorf("min int64 = %d", m.Minor())
	}
}

func TestArithmeticOverflowPanics(t *testing.T) {
	max, min := money.New(math.MaxInt64, usd), money.New(math.MinInt64, usd)
	mustPanicWith(t, money.ErrOutOfRange, func() { max.Add(money.New(1, usd)) })
	mustPanicWith(t, money.ErrOutOfRange, func() { min.Add(money.New(-1, usd)) })
	mustPanicWith(t, money.ErrOutOfRange, func() { min.Sub(money.New(1, usd)) })
	mustPanicWith(t, money.ErrOutOfRange, func() { max.Sub(money.New(-1, usd)) })
	mustPanicWith(t, money.ErrOutOfRange, func() { min.Neg() })
	mustPanicWith(t, money.ErrOutOfRange, func() { min.Abs() })

	// Right up to the edge is fine, and cancelling out is not an overflow.
	if got := max.Add(money.New(-1, usd)).Add(money.New(1, usd)); got.Minor() != math.MaxInt64 {
		t.Errorf("max-1+1 = %d", got.Minor())
	}
	if got := max.Add(min); got.Minor() != -1 {
		t.Errorf("max+min = %d", got.Minor())
	}
	if got := min.Sub(min); !got.IsZero() {
		t.Errorf("min-min = %d", got.Minor())
	}
}

func TestSumOfManyMaximaStaysExactAndComparable(t *testing.T) {
	moneytest.ForEach(t, func(t *testing.T, c moneytest.Currency) {
		unit := c.Unit(1)
		// A few thousand maximal rows is far beyond real data and still fits.
		total := money.Zero(unit)
		for i := 0; i < 5000; i++ {
			total = total.Add(money.New(money.MaxMinor, unit))
		}
		if total.Minor() != 5000*money.MaxMinor {
			t.Fatalf("total = %d", total.Minor())
		}
		if total.Cmp(money.New(money.MaxMinor, unit)) <= 0 || !total.IsPositive() {
			t.Fatal("ordering is wrong")
		}
		if got := total.Sub(total); !got.IsZero() {
			t.Fatalf("x-x = %d", got.Minor())
		}
	})
}

func TestMixedCurrenciesPanicForEveryPair(t *testing.T) {
	moneytest.ForEachPair(t, func(t *testing.T, from, to moneytest.Currency) {
		a, b := money.New(1, from.Unit(1)), money.New(1, to.Unit(2)) // different ids
		for name, op := range map[string]func(){
			"add": func() { a.Add(b) }, "sub": func() { a.Sub(b) }, "cmp": func() { a.Cmp(b) },
		} {
			func() {
				defer func() {
					if recover() == nil {
						t.Errorf("%s of %v and %v did not panic", name, from, to)
					}
				}()
				op()
			}()
		}
		if a.Equal(b) {
			t.Errorf("%v equals %v", a, b)
		}
	})
}

func TestEqualRequiresSameCurrency(t *testing.T) {
	if money.New(100, usd).Equal(money.New(100, jpy)) {
		t.Error("same minor units in different currencies are not equal")
	}
	if !money.New(100, usd).Equal(money.New(100, usd)) {
		t.Error("equal amounts reported unequal")
	}
}

func TestMarshalJSONIsACanonicalNumber(t *testing.T) {
	moneytest.ForEach(t, func(t *testing.T, c moneytest.Currency) {
		moneytest.ForEachAmount(t, moneytest.Valid(c.Decimals), func(t *testing.T, a moneytest.Amount) {
			m := money.New(a.Minor, c.Unit(1))
			out, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			want := decimal.New(a.Minor, -int32(c.Decimals)).String()
			if string(out) != want {
				t.Fatalf("json = %s, want %s", out, want)
			}
			// Plain digits only: no quotes, no exponent, so a client's number
			// parser reads it without surprises.
			var back decimal.Decimal
			if err := json.Unmarshal(out, &back); err != nil || !back.Equal(m.Decimal()) {
				t.Fatalf("does not round trip: %v, %v", back, err)
			}
		})
	})
}

func TestMaximumIsAnExactIntegerForJavaScript(t *testing.T) {
	if float64(money.MaxMinor) >= 1<<53 {
		t.Fatalf("MaxMinor %d is not below 2^53", money.MaxMinor)
	}
	if float64(money.MaxMinor)+1 != float64(money.MaxMinor+1) {
		t.Fatal("MaxMinor+1 is not exactly representable as a float64")
	}
}

func TestNullVariants(t *testing.T) {
	if money.FromNullMinor(sql.NullInt64{}, usd) != nil {
		t.Fatal("null should map to nil")
	}
	d := dec("1.25")
	m := money.FromNullDecimal(&d, usd)
	if n := money.ToNullMinor(m); !n.Valid || n.Int64 != 125 {
		t.Fatalf("got %+v", n)
	}
	if money.ToNullMinor(nil).Valid {
		t.Fatal("nil should map to invalid")
	}
	if money.FromNullDecimal(nil, usd) != nil {
		t.Fatal("nil should stay nil")
	}
	if got, err := money.FromNullInput(nil, usd); got != nil || err != nil {
		t.Fatalf("nil input = %v, %v", got, err)
	}
	huge := dec("1e30")
	if _, err := money.FromNullInput(&huge, btc); !errors.Is(err, money.ErrOutOfRange) {
		t.Fatalf("huge optional input = %v", err)
	}
}
