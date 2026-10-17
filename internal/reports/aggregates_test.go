package reports

import (
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
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

func TestReportTotalsFoldCurrenciesIntoBase(t *testing.T) {
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
		reports := Service{DB: e.db}

		// Everything is folded unrounded and rounded once.
		want := money.FromDecimal(exact, baseUnit)
		got := reports.TxSummary(e.ctx, janRange, "expense")
		if total := got.Total; !total.Equal(want) {
			t.Fatalf("report total = %s (%+v), want %s", total, total.Unit(), want)
		}
		sum := (ledger.Transactions{DB: e.db}).Summary(e.ctx, false)
		if !sum.Expense.Equal(want) {
			t.Fatalf("transaction summary = %s, want %s", sum.Expense, want)
		}
		if net := reports.Overview(e.ctx, janRange).NetCashFlow.Value; !net.Equal(want.Neg()) {
			t.Fatalf("net cash flow = %s, want %s", net, want.Neg())
		}
		// Per-day and per-category views agree with the grand total.
		if days := reports.Heatmap(e.ctx, janRange).Max; !days.Equal(want) {
			t.Fatalf("busiest day = %s, want %s", days, want)
		}
	})
}

func TestEachCurrencyConvertsRoundsToBase(t *testing.T) {
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
					created, terr := e.txs.Create(
						e.ctx,
						ledger.TxInput{Type: "expense", AccountID: e.accts[c.Code].ID, Amount: a.Input, Date: &date},
					)
					if terr != nil {
						t.Fatal(terr)
					}
					defer func() { _ = e.txs.Delete(e.ctx, created.ID) }()
					if err != nil {
						return // the base amount would be out of range: nothing sensible to compare
					}
					got := (Service{DB: e.db}).TxSummary(e.ctx, janRange, "expense").Total
					if !got.Equal(want) {
						t.Fatalf("%s %s is %s in the base, want %s", a.Name, a.Input, got, want)
					}
				})
			})
		}
	})
}

func TestNetWorthSummaryConvertsEachBalance(t *testing.T) {
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
		nw := (Service{DB: e.db}).NetWorth(e.ctx, Filter{}).Current
		if !nw.Equal(want) {
			t.Fatalf("net worth = %s, want %s", nw, want)
		}
		baseCur := e.stored(base)
		if total := (ledger.Accounts{DB: e.db}).Summary(e.ctx, &baseCur).Total; !total.Equal(want) {
			t.Fatalf("accounts summary = %s, want %s", total, want)
		}
	})
}
