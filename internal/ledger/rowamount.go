package ledger

import (
	"slices"

	"github.com/shopspring/decimal"

	"savvy-go/internal/money"
)

// CurrencyAmount is an amount with the currency it is in, rates included.
type CurrencyAmount struct {
	Currency Currency
	Amount   money.Money
}

// RowAmount is the one figure a list row shows. Main is in the first currency
// of Totals; when there are several, the others are converted into it at the
// current rates (Converted), and Totals keep the exact amounts. It is for
// display: nothing that decides or books money reads it.
type RowAmount struct {
	Main      CurrencyAmount
	Totals    []CurrencyAmount
	Converted bool
}

// DayTotal is what one day netted per currency, largest first by value in the
// base currency. Currencies are never added together.
type DayTotal struct {
	Date   string
	Totals []CurrencyAmount
}

// RowAmount is what was spent or earned, from the lines, unsigned. Without
// lines (a debt payment) it is what the payments on ordinary accounts net to.
// It is false when there is nothing to show or the conversion does not fit.
func (t Transaction) RowAmount() (RowAmount, bool) {
	if spent := t.spent(); len(spent) > 0 {
		return rowAmountOf(abs(spent))
	}
	return rowAmountOf(abs(sumBy(t.ordinaryPayments(func(money.Money) bool { return true }))))
}

// ListAmounts is the amount a list row shows and, for a transfer, the amount
// received. A transfer's main figure is what left its source accounts.
func (t Transaction) ListAmounts() (main, received *RowAmount) {
	if t.Type == "transfer" {
		sent := t.side(func(m money.Money) bool { return m.IsNegative() })
		got := t.side(func(m money.Money) bool { return m.IsPositive() })
		if sent != nil {
			return sent, got
		}
		main = got
	}
	if main == nil {
		if r, ok := t.RowAmount(); ok {
			main = &r
		}
	}
	return main, nil
}

func (t Transaction) side(keep func(money.Money) bool) *RowAmount {
	r, ok := rowAmountOf(abs(sumBy(t.ordinaryPayments(keep))))
	if !ok {
		return nil
	}
	return &r
}

func (t Transaction) ordinaryPayments(keep func(money.Money) bool) []CurrencyAmount {
	var out []CurrencyAmount
	for _, p := range t.Payments {
		if p.Account == nil || p.Account.Currency == nil || p.Account.IsService || !keep(p.Amount) {
			continue
		}
		out = append(out, CurrencyAmount{Currency: *p.Account.Currency, Amount: p.Amount})
	}
	return out
}

// spent is the lines per currency, in order of first appearance. A line is in
// one of the currencies of the payment accounts; one that is not is skipped.
func (t Transaction) spent() []CurrencyAmount {
	var entries []CurrencyAmount
	for _, l := range t.Lines {
		for _, p := range t.Payments {
			if p.Account != nil && p.Account.Currency != nil && p.Account.Currency.Unit() == l.Amount.Unit() {
				entries = append(entries, CurrencyAmount{Currency: *p.Account.Currency, Amount: l.Amount})
				break
			}
		}
	}
	return sumBy(entries)
}

func sumBy(entries []CurrencyAmount) []CurrencyAmount {
	var out []CurrencyAmount
	for _, e := range entries {
		i := slices.IndexFunc(out, func(o CurrencyAmount) bool { return o.Currency.ID == e.Currency.ID })
		if i < 0 {
			out = append(out, e)
			continue
		}
		out[i].Amount = out[i].Amount.Add(e.Amount)
	}
	return out
}

func abs(in []CurrencyAmount) []CurrencyAmount {
	for i := range in {
		in[i].Amount = in[i].Amount.Abs()
	}
	return in
}

func rowAmountOf(totals []CurrencyAmount) (RowAmount, bool) {
	if len(totals) == 0 {
		return RowAmount{}, false
	}
	first := totals[0]
	out := RowAmount{Main: first, Totals: totals, Converted: len(totals) > 1}
	for _, t := range totals[1:] {
		converted, err := TryConvert(t.Amount, t.Currency, first.Currency)
		if err != nil {
			return RowAmount{}, false
		}
		out.Main.Amount = out.Main.Amount.Add(converted)
	}
	return out, true
}

// signOf is how a type moves the day: +1 for money in, -1 for money out, 0
// for none. A transfer is handled apart, by its fee lines.
func signOf(typ string) int {
	switch typ {
	case "income", "debt_collection", "debt_borrow", "transfer_in":
		return 1
	case "expense", "debt_payment", "debt_lend", "transfer_out":
		return -1
	}
	return 0
}

// DayTotals nets the transactions per date and currency: income less expense,
// a transfer counting only its fee lines, skipped ones not at all. Days are in
// order of first appearance; a transaction without a date is left out.
func DayTotals(txs []Transaction) []DayTotal {
	var days []DayTotal
	for _, t := range txs {
		if t.Status == "skipped" || t.Date == nil {
			continue
		}
		var parts []CurrencyAmount
		sign := signOf(t.Type)
		if t.Type == "transfer" {
			sign, parts = -1, abs(t.spent())
		} else if r, ok := t.RowAmount(); ok {
			parts = r.Totals
		}
		if sign == 0 || len(parts) == 0 {
			continue
		}
		i := slices.IndexFunc(days, func(d DayTotal) bool { return d.Date == *t.Date })
		if i < 0 {
			days = append(days, DayTotal{Date: *t.Date})
			i = len(days) - 1
		}
		for _, p := range parts {
			if sign < 0 {
				p.Amount = p.Amount.Neg()
			}
			days[i].Totals = sumBy(append(days[i].Totals, p))
		}
	}
	for i := range days {
		slices.SortStableFunc(days[i].Totals, func(a, b CurrencyAmount) int { return baseWeight(b).Cmp(baseWeight(a)) })
	}
	return days
}

// baseWeight ranks an amount by its absolute value in the base currency.
func baseWeight(c CurrencyAmount) decimal.Decimal {
	return c.Currency.ConvertToBase(c.Amount.Decimal().Abs())
}
