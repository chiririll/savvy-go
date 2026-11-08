package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
)

// richLine is one line of a richTx: an amount for a category, in the currency
// of the account named by cur (the first payment's when empty).
type richLine struct {
	category, cur string
	amount        string
	note          string
}

// richPay is money moving on an account; negative when it leaves.
type richPay struct{ account, amount string }

// richTx is a transaction the old flat model could not hold: several payments,
// lines in different categories or currencies, notes written as markdown.
// daysAgo is negative for a day ahead, which makes the transaction pending.
type richTx struct {
	typ, desc string
	daysAgo   int
	pays      []richPay
	lines     []richLine
	tags      []string
}

// richTransactions are placed relative to today and use no random draws, so
// the rest of the demo is the same with or without them.
var richTransactions = []richTx{
	{
		typ: "expense", desc: "Weekly shopping — Whole Foods and Target", daysAgo: 2,
		pays: []richPay{{"credit", "-96.40"}},
		lines: []richLine{
			{"#GROCERIES", "", "64.40", "- Whole milk ×2\n- Sourdough bread\n- Free-range eggs\n- Chicken breast 1.2 kg\n- Baby spinach"},
			{"#HOUSING", "", "18.00", "- Dish soap\n- Paper towels\n- Light bulbs ×4"},
			{"#HEALTH", "", "14.00", "- Vitamin D\n- Ibuprofen\n\nReceipt: [Target order 4821](https://example.com/receipts/4821)"},
		},
		tags: []string{"#ESSENTIAL"},
	},
	{
		typ: "expense", desc: "Costco run — cash, debit and credit card", daysAgo: 11,
		pays: []richPay{{"cash", "-40.00"}, {"checking", "-60.00"}, {"credit", "-85.50"}},
		lines: []richLine{
			{"#GROCERIES", "", "120.50", "- [x] Olive oil 3 L\n- [x] Coffee beans 1 kg\n- [x] Pasta ×6\n- [ ] Salmon fillets (sold out)"},
			{"#HOUSING", "", "40.00", "- [x] Laundry detergent\n- [x] Trash bags"},
			{"#CLOTHES_CARE", "", "25.00", "- [x] Socks, 6 pairs"},
		},
	},
	{
		typ: "expense", desc: "Lisbon weekend — paid in euros, change in dollars", daysAgo: 20,
		pays: []richPay{{"eur", "-120.00"}, {"cash", "10.80"}},
		lines: []richLine{
			{"#TRAVEL", "eur", "70.00", "- Tram day pass ×2\n- Castle tickets\n- Pastéis de nata"},
			{"#EATING_OUT", "eur", "40.00", "Dinner in Alfama"},
		},
		tags: []string{"#VACATION"},
	},
	{
		typ: "income", desc: "Year-end bonus — Acme Corp", daysAgo: 8,
		pays:  []richPay{{"checking", "3200.00"}, {"savings", "800.00"}},
		lines: []richLine{{"#SALARY", "", "4000.00", "Paid out in two parts: 80% to checking, 20% straight to savings."}},
	},
	{
		typ: "transfer", desc: "Wire to Revolut EUR", daysAgo: 15,
		pays:  []richPay{{"checking", "-515.00"}, {"eur", "462.96"}},
		lines: []richLine{{"#OTHER", "", "15.00", "Wire fee"}},
	},
	{
		// A shopping list: nothing is bought yet, so nothing moves.
		typ: "expense", desc: "Weekend shopping list", daysAgo: -2,
		pays: []richPay{{"checking", "-102.00"}},
		lines: []richLine{
			{"#GROCERIES", "", "78.00", "- [x] Bananas\n- [x] Greek yogurt\n- [ ] Cheddar cheese\n- [ ] Roma tomatoes\n- [ ] Orange juice\n- [ ] Granola"},
			{"#HOUSING", "", "24.00", "- [ ] Shelf brackets ×4\n- [ ] Paint roller"},
		},
	},
}

// seedRich adds richTransactions to the Demo space.
func (s *seeder) seedRich(accounts map[string]*ledger.Account, expenses, incomes []ledger.Category, tags []ledger.Tag) error {
	cats := append(append([]ledger.Category{}, expenses...), incomes...)
	return createRich(s.ctx, s.txs, s.now, richTransactions, accounts, cats, tags)
}

// createRich adds list to the space, dated relative to now. Each one must agree
// with itself (the ledger reports no difference between payments and lines), so
// a mistake in a table fails the seeding instead of showing odd data. One that
// needs an account the space lacks is left out.
func createRich(
	ctx context.Context,
	txs ledger.Transactions,
	now time.Time,
	list []richTx,
	accounts map[string]*ledger.Account,
	cats []ledger.Category,
	tags []ledger.Tag,
) error {
	for _, t := range list {
		in, ok, err := richInput(t, now, accounts, cats, tags)
		if err != nil {
			return fmt.Errorf("%s: %w", t.desc, err)
		}
		if !ok {
			continue
		}
		created, err := txs.Create(ctx, in)
		if err != nil {
			return fmt.Errorf("%s: %w", t.desc, err)
		}
		if created.Difference != nil && !created.Difference.IsZero() {
			return fmt.Errorf("%s: payments and lines differ by %s", t.desc, created.Difference)
		}
	}
	return nil
}

// richInput builds the input of t; false when an account it needs is absent.
func richInput(
	t richTx, now time.Time, accounts map[string]*ledger.Account, cats []ledger.Category, tags []ledger.Tag,
) (ledger.TxInput, bool, error) {
	in := ledger.TxInput{Type: t.typ, Description: &t.desc}
	for _, p := range t.pays {
		a := accounts[p.account]
		if a == nil {
			return in, false, nil
		}
		amount, err := decimal.NewFromString(p.amount)
		if err != nil {
			return in, false, err
		}
		in.Payments = append(in.Payments, ledger.PaymentInput{AccountID: a.ID, Amount: &amount})
	}
	for _, l := range t.lines {
		if l.cur != "" && accounts[l.cur] == nil {
			return in, false, nil
		}
		amount, err := decimal.NewFromString(l.amount)
		if err != nil {
			return in, false, err
		}
		line := ledger.LineInput{Amount: amount}
		if c := catByName(cats, l.category); c != nil {
			line.CategoryID = &c.ID
		}
		cur := accounts[t.pays[0].account]
		if l.cur != "" {
			cur = accounts[l.cur]
		}
		line.CurrencyID = cur.CurrencyID
		if l.note != "" {
			note := l.note
			line.Note = &note
		}
		in.Lines = append(in.Lines, line)
	}
	for _, name := range t.tags {
		if tag := tagByName(tags, name); tag != nil {
			in.TagIDs = append(in.TagIDs, tag.ID)
		}
	}
	day := now.AddDate(0, 0, -t.daysAgo).Format("2006-01-02")
	status := "confirmed"
	if t.daysAgo < 0 {
		status = "pending"
	}
	in.Date, in.Status = &day, &status
	return in, true, nil
}
