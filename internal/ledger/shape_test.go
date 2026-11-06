package ledger

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/money"
)

var (
	eur = Currency{ID: 1, Code: "EUR", Decimals: 2, Rate: decimal.NewFromInt(1), IsBase: true}
	rsd = Currency{ID: 2, Code: "RSD", Decimals: 2, Rate: decimal.RequireFromString("0.01")}
)

func acct(id int64, typ string, c Currency) Account {
	cur := c
	return Account{ID: id, Type: typ, IsService: isServiceType(typ), CurrencyID: c.ID, Currency: &cur}
}

func pay(a Account, minor int64) shapePayment {
	return shapePayment{Account: a, Amount: money.New(minor, a.Currency.Unit())}
}

func line(c Currency, cat string, minor int64) shapeLine {
	l := shapeLine{Currency: c, Amount: money.New(minor, c.Unit())}
	if cat != "" {
		l.Category = &Category{ID: 1, Type: cat}
	}
	return l
}

func TestShapeCheck(t *testing.T) {
	card, cash := acct(1, "bank", eur), acct(2, "cash", rsd)
	debt, space := acct(3, "debt", eur), acct(4, "space", eur)
	cases := []struct {
		name  string
		shape shape
		want  error
	}{
		{"expense", shape{Type: "expense", Payments: []shapePayment{pay(card, -1000)},
			Lines: []shapeLine{line(eur, "expense", 1000)}}, nil},
		{"expense paid in two currencies with change", shape{Type: "expense",
			Payments: []shapePayment{pay(card, -2000), pay(cash, 100000)},
			Lines:    []shapeLine{line(eur, "expense", 1000)}}, nil},
		{"expense with no payment", shape{Type: "expense", Lines: []shapeLine{line(eur, "expense", 1)}}, ErrNoPayments},
		{"expense with no line", shape{Type: "expense", Payments: []shapePayment{pay(card, -1000)}}, ErrNoLines},
		{"one account twice", shape{Type: "expense", Payments: []shapePayment{pay(card, -1000), pay(card, 100)},
			Lines: []shapeLine{line(eur, "expense", 900)}}, ErrDuplicateAccount},
		{"payment of zero", shape{Type: "expense", Payments: []shapePayment{pay(card, 0)},
			Lines: []shapeLine{line(eur, "expense", 1)}}, ErrZeroPayment},
		{"expense that gains", shape{Type: "expense", Payments: []shapePayment{pay(card, 1000)},
			Lines: []shapeLine{line(eur, "expense", 1000)}}, ErrNetSign},
		{"income that loses", shape{Type: "income", Payments: []shapePayment{pay(card, -1000)},
			Lines: []shapeLine{line(eur, "income", 1000)}}, ErrNetSign},
		{"income", shape{Type: "income", Payments: []shapePayment{pay(card, 1000)},
			Lines: []shapeLine{line(eur, "income", 1000)}}, nil},
		{"uncategorised line", shape{Type: "expense", Payments: []shapePayment{pay(card, -1000)},
			Lines: []shapeLine{line(eur, "", 1000)}}, nil},
		{"income category on an expense", shape{Type: "expense", Payments: []shapePayment{pay(card, -1000)},
			Lines: []shapeLine{line(eur, "income", 1000)}}, ErrLineCategory},
		{"line in a currency no account has", shape{Type: "expense", Payments: []shapePayment{pay(card, -1000)},
			Lines: []shapeLine{line(rsd, "expense", 1000)}}, ErrLineCurrency},
		{"line of zero", shape{Type: "expense", Payments: []shapePayment{pay(card, -1000)},
			Lines: []shapeLine{line(eur, "expense", 0)}}, ErrLineAmount},
		{"transfer across currencies with a fee", shape{Type: "transfer",
			Payments: []shapePayment{pay(card, -10000), pay(cash, 1170000-20)},
			Lines:    []shapeLine{line(eur, "expense", 2)}}, nil},
		{"transfer that only takes", shape{Type: "transfer", Payments: []shapePayment{pay(card, -1000)}}, ErrTransferSigns},
		{"transfer fee with an income category", shape{Type: "transfer",
			Payments: []shapePayment{pay(card, -1000), pay(cash, 1000)},
			Lines:    []shapeLine{line(eur, "income", 1)}}, ErrLineCategory},
		{"payment on a debt account by hand", shape{Type: "transfer",
			Payments: []shapePayment{pay(card, -1000), pay(debt, 1000)}}, ErrServiceAccount},
		{"payment on a space account by hand", shape{Type: "expense",
			Payments: []shapePayment{pay(space, -1000)}, Lines: []shapeLine{line(eur, "expense", 1000)}}, ErrServiceAccount},
		{"debt lent by the debts module", shape{Type: "debt_lend", Owner: ReasonDebt,
			Payments: []shapePayment{pay(card, -1000), pay(debt, 1000)}}, nil},
		{"debt lent with the wrong signs", shape{Type: "debt_lend", Owner: ReasonDebt,
			Payments: []shapePayment{pay(card, 1000), pay(debt, -1000)}}, ErrDebtShape},
		{"debt without a debt account", shape{Type: "debt_payment", Owner: ReasonDebt,
			Payments: []shapePayment{pay(card, -1000), pay(cash, 1000)}}, ErrDebtShape},
		{"debt with a line", shape{Type: "debt_payment", Owner: ReasonDebt,
			Payments: []shapePayment{pay(card, -1000), pay(debt, 1000)}, Lines: []shapeLine{line(eur, "expense", 1000)}},
			ErrDebtShape},
		{"debt type by hand", shape{Type: "debt_lend",
			Payments: []shapePayment{pay(card, -1000), pay(debt, 1000)}}, ErrTxType},
		{"transfer out written by transfers", shape{Type: "transfer_out", Owner: ReasonSpaceTransfer,
			Payments: []shapePayment{pay(card, -1000), pay(space, 1000)}}, nil},
		{"transfer out without a space payment", shape{Type: "transfer_out", Owner: ReasonSpaceTransfer,
			Payments: []shapePayment{pay(card, -1000)}}, ErrSpaceShape},
		{"transfer out by hand", shape{Type: "transfer_out",
			Payments: []shapePayment{pay(card, -1000), pay(space, 1000)}}, ErrTxType},
		{"shared purchase paid from the space account", shape{Type: "expense", Owner: ReasonSpaceTransfer,
			Payments: []shapePayment{pay(space, -4000)}, Lines: []shapeLine{line(eur, "expense", 4000)}}, nil},
		{"two space payments", shape{Type: "expense", Owner: ReasonSpaceTransfer,
			Payments: []shapePayment{pay(space, -4000), pay(acct(5, "space", rsd), -100)},
			Lines:    []shapeLine{line(eur, "expense", 4000)}}, ErrSpaceShape},
		{"template with a balancing payment", shape{Type: "transfer", Template: true,
			Payments: []shapePayment{pay(card, -1000), {Account: cash, Balancing: true}}}, nil},
		{"template with two balancing payments", shape{Type: "transfer", Template: true,
			Payments: []shapePayment{{Account: card, Balancing: true}, {Account: cash, Balancing: true}}}, ErrManyBalancing},
		{"balancing payment outside a template", shape{Type: "transfer",
			Payments: []shapePayment{pay(card, -1000), {Account: cash, Balancing: true}}}, ErrBalancing},
		{"template with a space payment", shape{Type: "expense", Template: true, Owner: ReasonSpaceTransfer,
			Payments: []shapePayment{pay(space, -1000)}, Lines: []shapeLine{line(eur, "expense", 1000)}}, ErrSpaceShape},
	}
	for _, c := range cases {
		if got := c.shape.check(); !errors.Is(got, c.want) {
			t.Errorf("%s: check() = %v, want %v", c.name, got, c.want)
		}
	}
}

// The soft check converts payments and lines to the base currency at the
// current rates and only reports the difference, never blocks.
func TestShapeDifference(t *testing.T) {
	card, cash := acct(1, "bank", eur), acct(2, "cash", rsd)
	cases := []struct {
		name  string
		shape shape
		want  int64 // minor units of EUR
	}{
		{"matching", shape{Type: "expense", Payments: []shapePayment{pay(card, -1000)},
			Lines: []shapeLine{line(eur, "expense", 1000)}}, 0},
		{"change in another currency", shape{Type: "expense",
			Payments: []shapePayment{pay(card, -2000), pay(cash, 100000)}, // 1000.00 RSD = 10.00 EUR back
			Lines:    []shapeLine{line(eur, "expense", 1000)}}, 0},
		{"paid more than the lines", shape{Type: "expense", Payments: []shapePayment{pay(card, -1500)},
			Lines: []shapeLine{line(eur, "expense", 1000)}}, 500},
		{"lines more than paid", shape{Type: "income", Payments: []shapePayment{pay(card, 900)},
			Lines: []shapeLine{line(eur, "income", 1000)}}, -100},
		{"transfer with no fee", shape{Type: "transfer",
			Payments: []shapePayment{pay(card, -1000), pay(cash, 100000)}}, 0},
		{"transfer fee as a line", shape{Type: "transfer",
			Payments: []shapePayment{pay(card, -1000), pay(cash, 98000)}, // 0.20 EUR lost
			Lines:    []shapeLine{line(eur, "expense", 20)}}, 0},
	}
	for _, c := range cases {
		got, ok := c.shape.difference(eur)
		if !ok || got.Minor() != c.want {
			t.Errorf("%s: difference = %s, want %d minor", c.name, got, c.want)
		}
	}
}

// The soft check never fails a save: an amount that does not fit in the base
// currency is simply not reported.
func TestShapeDifferenceSkipsWhatDoesNotFit(t *testing.T) {
	big := Currency{ID: 3, Code: "BIG", Decimals: 0, Rate: decimal.NewFromInt(1_000_000)}
	a := acct(9, "bank", big)
	s := shape{Type: "expense", Payments: []shapePayment{pay(a, -money.MaxMinor)},
		Lines: []shapeLine{line(big, "expense", 1)}}
	if got, ok := s.difference(eur); ok {
		t.Errorf("difference = %s, want none", got)
	}
}
