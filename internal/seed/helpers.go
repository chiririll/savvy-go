package seed

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	ledgerrepo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/ledger"
)

func (s *seeder) addTx(
	typ string,
	accountID int64,
	catID *int64,
	amount float64,
	desc string,
	date time.Time,
	toID *int64,
	toAmt *float64,
	tagIDs []int64,
) error {
	if !s.inRange(date) {
		return nil
	}
	d := date.Format("2006-01-02")
	if d > s.now.Format("2006-01-02") {
		return nil
	}
	status := "confirmed"
	in, err := s.input(typ, accountID, catID, amount, toID, toAmt)
	if err != nil {
		return fmt.Errorf("%s %s: %w", typ, desc, err)
	}
	in.Description, in.Date, in.Status, in.TagIDs = &desc, &d, &status, tagIDs
	txs := s.txs
	if strings.HasPrefix(typ, "debt_") {
		txs = txs.As(ledger.ReasonDebt)
	}
	// Add, not Create: reading each transaction back costs more with every one
	// added, which is most of the time of a long history.
	id, err := txs.Add(s.ctx, in)
	if err != nil {
		return fmt.Errorf("%s %s: %w", typ, desc, err)
	}
	if typ == "expense" && amount >= 2 {
		name := ""
		if catID != nil {
			name = s.catNames[*catID]
		}
		s.expenses = append(s.expenses, expenseCand{id: id, amount: amount, catName: name, catID: catID})
	}
	return nil
}

// input builds the payments (and the line, for an income or expense) of a
// transaction of the old flat kind: an amount on an account and, for a
// transfer or a debt, the account it went to.
func (s *seeder) input(typ string, accountID int64, catID *int64, amount float64, toID *int64, toAmt *float64) (ledger.TxInput, error) {
	value := decimal.NewFromFloat(amount)
	switch typ {
	case "income", "expense":
		paid := value
		if typ == "expense" {
			paid = value.Neg()
		}
		return ledger.TxInput{
			Type:     typ,
			Payments: []ledger.PaymentInput{{AccountID: accountID, Amount: &paid}},
			Lines:    []ledger.LineInput{{CategoryID: catID, Amount: value}},
		}, nil
	case "transfer":
		if toID == nil {
			return ledger.TxInput{}, fmt.Errorf("a transfer needs a destination")
		}
		if toAmt == nil {
			return s.txs.ConvertedTransfer(s.ctx, accountID, *toID, value)
		}
		out, in := value.Neg(), decimal.NewFromFloat(*toAmt)
		return ledger.TxInput{Type: typ, Payments: []ledger.PaymentInput{
			{AccountID: accountID, Amount: &out}, {AccountID: *toID, Amount: &in},
		}}, nil
	}
	// A debt transaction: lending and paying add to the debt account and take
	// from the ordinary one; borrowing and collecting do the reverse.
	if toID == nil {
		return ledger.TxInput{}, fmt.Errorf("a debt transaction needs the debt")
	}
	onDebt, onAccount := value, value.Neg()
	if typ == "debt_borrow" || typ == "debt_collection" {
		onDebt, onAccount = value.Neg(), value
	}
	return ledger.TxInput{Type: typ, Payments: []ledger.PaymentInput{
		{AccountID: accountID, Amount: &onAccount}, {AccountID: *toID, Amount: &onDebt},
	}}, nil
}

func (s *seeder) countTx() int {
	return ledgerrepo.Repo{DB: s.db}.CountTransactions(s.ctx)
}

func (s *seeder) inRange(d time.Time) bool {
	day := d.Format("2006-01-02")
	return day >= s.start.Format("2006-01-02") && day <= s.end.Format("2006-01-02")
}

func (s *seeder) mtRand(lo, hi int) int {
	if hi < lo {
		lo, hi = hi, lo
	}
	return lo + s.rng.Intn(hi-lo+1)
}

func (s *seeder) merchantFor(category string) string {
	list := merchants[category]
	if len(list) == 0 {
		return "General Store"
	}
	return list[s.rng.Intn(len(list))]
}

func catByName(cats []ledger.Category, name string) *ledger.Category {
	for i := range cats {
		if cats[i].Name == name {
			return &cats[i]
		}
	}
	return nil
}

func tagByName(tags []ledger.Tag, name string) *ledger.Tag {
	for i := range tags {
		if tags[i].Name == name {
			return &tags[i]
		}
	}
	return nil
}

func strPtr(s string) *string { return &s }

func inList(v string, opts ...string) bool {
	for _, o := range opts {
		if v == o {
			return true
		}
	}
	return false
}
