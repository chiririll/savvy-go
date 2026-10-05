package domain

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

func TestReportTotalsFoldEveryCurrencyIntoEveryBase(t *testing.T) {
	forEachBase(t, func(t *testing.T, base moneytest.Currency) {
		e := newMatrixEnv(t, base)
		baseUnit := e.stored(base).Unit()
		exact := decimal.Zero
		for _, c := range fitting(base) {
			if _, err := e.tx(t, "expense", c, typical(c)); err != nil {
				t.Fatal(err)
			}
			exact = exact.Add(asBase(typical(c), c, base))
		}
		reports := Reports{DB: e.db}

		// Everything is folded unrounded and rounded once.
		want := money.FromDecimal(exact, baseUnit)
		got := reports.TxSummary(e.ctx, janRange, "expense")
		if total := asMoney(t, got["total"]); !total.Equal(want) {
			t.Fatalf("report total = %s (%+v), want %s", total, total.Unit(), want)
		}
		sum := (Transactions{DB: e.db}).Summary(e.ctx, false)
		if !sum.Expense.Equal(want) {
			t.Fatalf("transaction summary = %s, want %s", sum.Expense, want)
		}
		if net := asMoney(t, reports.Overview(e.ctx, janRange)["netCashFlow"].(map[string]any)["value"]); !net.Equal(want.Neg()) {
			t.Fatalf("net cash flow = %s, want %s", net, want.Neg())
		}
		// Per-day and per-category views agree with the grand total.
		if days := reports.Heatmap(e.ctx, janRange)["max"]; !asMoney(t, days).Equal(want) {
			t.Fatalf("busiest day = %s, want %s", days, want)
		}
	})
}

func TestEachCurrencyAloneIsConvertedAndRoundedIntoTheBase(t *testing.T) {
	forEachBase(t, func(t *testing.T, base moneytest.Currency) {
		for _, c := range moneytest.Currencies {
			t.Run(c.Code, func(t *testing.T) {
				e := newMatrixEnv(t, base)
				moneytest.ForEachAmount(t, moneytest.Valid(c.Decimals), func(t *testing.T, a moneytest.Amount) {
					if a.Minor == 0 {
						return
					}
					// What is converted is what was stored: the input rounded to the
					// currency's own minor unit first.
					stored := decimal.New(a.Minor, -int32(c.Decimals))
					want, err := money.FromInput(asBase(stored, c, base), e.stored(base).Unit())
					date := "2024-01-15"
					created, terr := e.txs.Create(e.ctx, TxInput{Type: "expense", AccountID: e.accts[c.Code].ID, Amount: a.Input, Date: &date})
					if terr != nil {
						t.Fatal(terr)
					}
					defer func() { _ = e.txs.Delete(e.ctx, created.ID) }()
					if err != nil {
						return // the base amount would be out of range: nothing sensible to compare
					}
					got := asMoney(t, (Reports{DB: e.db}).TxSummary(e.ctx, janRange, "expense")["total"])
					if !got.Equal(want) {
						t.Fatalf("%s %s is %s in the base, want %s", a.Name, a.Input, got, want)
					}
				})
			})
		}
	})
}

func TestNetWorthAndAccountSummaryConvertEachBalance(t *testing.T) {
	forEachBase(t, func(t *testing.T, base moneytest.Currency) {
		e := newMatrixEnv(t, base)
		baseUnit := e.stored(base).Unit()
		want := money.Zero(baseUnit)
		for _, c := range fitting(base) {
			if _, err := e.tx(t, "income", c, typical(c)); err != nil {
				t.Fatal(err)
			}
			// Balances are converted and rounded one account at a time.
			want = want.Add(money.FromDecimal(asBase(typical(c), c, base), baseUnit))
		}
		nw := asMoney(t, (Reports{DB: e.db}).NetWorth(e.ctx, ReportFilter{})["current"])
		if !nw.Equal(want) {
			t.Fatalf("net worth = %s, want %s", nw, want)
		}
		baseCur := e.stored(base)
		if total := (Accounts{DB: e.db}).Summary(e.ctx, &baseCur).Total; !total.Equal(want) {
			t.Fatalf("accounts summary = %s, want %s", total, want)
		}
	})
}

func TestBudgetSpentIsConvertedIntoTheBudgetCurrency(t *testing.T) {
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

func TestBudgetOfMaximumAmountHasNoOverflow(t *testing.T) {
	forEachBase(t, func(t *testing.T, c moneytest.Currency) {
		e := newMatrixEnv(t, moneytest.Currencies[0])
		cur := e.stored(c)
		max := decimal.New(money.MaxMinor, -int32(c.Decimals))
		budget, err := (Budgets{DB: e.db}).Create(e.ctx, BudgetInput{Name: "max", Amount: max, CurrencyID: &cur.ID, Period: "monthly", IsGlobal: ptrTo(true)})
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

func TestCurrencyDecimalsAndRatesAreRangeChecked(t *testing.T) {
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

func TestRateFromFloatKeepsSignificantDigits(t *testing.T) {
	for _, c := range []struct {
		in   float64
		want string
	}{
		{3e-9, "0.000000003"},
		{3e-13, "0.0000000000003"}, // a fixed 12 places would have made this zero
		{1.0 / 3, "0.333333333333"},
		{65000.123456789012345, "65000.1234568"}, // 12 significant digits
		{0.0067, "0.0067"},
		{2.38e-5, "0.0000238"},
	} {
		if got := rateFromFloat(c.in); !got.Equal(dec(c.want)) {
			t.Errorf("rateFromFloat(%v) = %s, want %s", c.in, got, c.want)
		}
	}
	if !rateFromFloat(0).IsZero() {
		t.Error("zero must stay zero so callers skip it")
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
