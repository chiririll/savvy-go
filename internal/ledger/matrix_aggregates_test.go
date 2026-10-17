package ledger

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/money"
	"savvy-go/internal/money/moneytest"
)

// typical is the amount, in major units, a currency's test transactions use:
// 12345 of its minor units.
func typical(c moneytest.Currency) decimal.Decimal { return decimal.New(12345, -int32(c.Decimals)) }

// asBase is amount of c expressed in base major units, unrounded.
func asBase(amount decimal.Decimal, c, base moneytest.Currency) decimal.Decimal {
	return moneytest.Converted(amount, c, base)
}

// fitting are the currencies whose typical amount stays well inside the range
// when expressed in base. A base such as a twelve-decimal token worth 3e-9
// dollars cannot hold the value of a bitcoin: its minor units run past
// money.MaxMinor, and a total over such amounts is out of range by nature.
func fitting(base moneytest.Currency) []moneytest.Currency {
	var out []moneytest.Currency
	for _, c := range moneytest.Currencies {
		// Leave room for the sum of every currency's amount.
		if moneytest.Fits(asBase(typical(c), c, base).Mul(decimal.NewFromInt(100)), base.Decimals) {
			out = append(out, c)
		}
	}
	return out
}

// forEachBase runs fn once per currency taken as the base of the database.
func forEachBase(t *testing.T, fn func(t *testing.T, base moneytest.Currency)) {
	t.Helper()
	moneytest.ForEach(t, fn)
}

func TestBudgetSpentConvertedIntoBudgetCurrency(t *testing.T) {
	forEachBase(t, func(t *testing.T, budgetCur moneytest.Currency) {
		e := newMatrixEnv(t, moneytest.Currencies[0])
		b := e.stored(budgetCur)
		exact := decimal.Zero
		for _, c := range fitting(budgetCur) {
			if _, err := e.tx(t, "expense", c, typical(c)); err != nil {
				t.Fatal(err)
			}
			exact = exact.Add(moneytest.Converted(typical(c), c, budgetCur))
		}
		budget, err := (Budgets{DB: e.db}).Create(e.ctx, BudgetInput{
			Name: "all", Amount: decimal.NewFromInt(1000), CurrencyID: &b.ID, Period: "one_time",
			StartDate: ptrTo("2024-01-01"), EndDate: ptrTo("2024-01-31"), IsGlobal: ptrTo(true),
		})
		if err != nil {
			t.Fatal(err)
		}
		want := money.FromDecimal(exact, b.Unit())
		got := budget.Progress.Spent
		if !got.Equal(want) {
			t.Fatalf("spent = %s (%+v), want %s in %+v", got, got.Unit(), want, b.Unit())
		}
		if got.Unit() != budget.Amount.Unit() {
			t.Fatalf("spent is in %+v, the budget in %+v", got.Unit(), budget.Amount.Unit())
		}
		if budget.Progress.Remaining.Unit() != b.Unit() {
			t.Fatalf("remaining is in %+v", budget.Progress.Remaining.Unit())
		}
	})
}

func TestBudgetMaximumAmountHasNoOverflow(t *testing.T) {
	forEachBase(t, func(t *testing.T, c moneytest.Currency) {
		e := newMatrixEnv(t, moneytest.Currencies[0])
		cur := e.stored(c)
		max := decimal.New(money.MaxMinor, -int32(c.Decimals))
		budget, err := (Budgets{DB: e.db}).Create(
			e.ctx,
			BudgetInput{Name: "max", Amount: max, CurrencyID: &cur.ID, Period: "monthly", IsGlobal: ptrTo(true)},
		)
		if err != nil {
			t.Fatal(err)
		}
		if budget.Amount.Minor() != money.MaxMinor || budget.Progress.IsExceeded || budget.Progress.Percent != 0 {
			t.Fatalf("budget = %+v / %+v", budget.Amount, budget.Progress)
		}
		if !budget.Progress.Remaining.Equal(budget.Amount) {
			t.Fatalf("remaining = %s, want the whole budget", budget.Progress.Remaining)
		}
	})
}

func TestCurrencyDecimalsRatesAreRangeChecked(t *testing.T) {
	e := newMatrixEnv(t, moneytest.Currencies[0])
	curs := Currencies{DB: e.db}
	mk := func(code string, decimals int, rate string) error {
		_, err := curs.Create(e.ctx, Currency{Code: code, Name: code, Symbol: code, Decimals: decimals, Rate: dec(rate)})
		return err
	}
	for _, c := range []struct {
		name     string
		decimals int
		rate     string
		want     error
	}{
		{"zero decimals", 0, "1", nil},
		{"twelve decimals", 12, "1", nil},
		{"thirteen decimals", 13, "1", ErrInvalidDecimals},
		{"smallest rate", 2, "0.000000000000001", nil},
		{"rate below the minimum", 2, "0.0000000000000009", ErrInvalidRate},
		{"largest rate", 2, "1000000000000000", nil},
		{"rate above the maximum", 2, "1000000000000001", ErrInvalidRate},
		{"negative rate", 2, "-1", ErrInvalidRate},
	} {
		code := "T" + string(rune('A'+len(c.name)%26)) + c.name[:2]
		if err := mk(code, c.decimals, c.rate); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", c.name, err, c.want)
		}
	}
}

// Choosing another base currency rescales every rate; conversions must come
// out the same afterwards, for every currency and every amount.
func TestChangingTheBaseKeepsEveryConversion(t *testing.T) {
	moneytest.ForEach(t, func(t *testing.T, newBase moneytest.Currency) {
		e := newMatrixEnv(t, moneytest.Currencies[0])
		curs := Currencies{DB: e.db}
		type key struct{ from, to string }
		convert := func() map[key]string {
			out := map[key]string{}
			for _, from := range moneytest.Currencies {
				for _, to := range moneytest.Currencies {
					f, _ := curs.ByCode(e.ctx, from.Code)
					tt, _ := curs.ByCode(e.ctx, to.Code)
					m, err := TryConvert(money.FromDecimal(typical(from), f.Unit()), *f, *tt)
					if err != nil {
						out[key{from.Code, to.Code}] = "out of range"
						continue
					}
					out[key{from.Code, to.Code}] = m.String()
				}
			}
			return out
		}
		before := convert()
		changed, err := curs.SetBase(e.ctx, e.curs[newBase.Code].ID)
		if err != nil {
			t.Fatal(err)
		}
		if !changed.IsBase || !changed.Rate.Equal(one) {
			t.Fatalf("new base = %+v", changed)
		}
		for k, was := range before {
			if got := convert()[k]; got != was {
				t.Errorf("%s->%s: %s before, %s after the base changed", k.from, k.to, was, got)
			}
		}
	})
}

func ptrTo[T any](v T) *T { return &v }
