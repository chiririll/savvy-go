package seed

import (
	"fmt"
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
	in := ledger.TxInput{
		Type: typ, AccountID: accountID, ToAccountID: toID, CategoryID: catID,
		Amount: decimal.NewFromFloat(amount), ToAmount: floatPtrDecimal(toAmt),
		Description: &desc, Date: &d, Status: &status, TagIDs: tagIDs,
	}
	created, err := s.txs.Create(s.ctx, in)
	if err != nil {
		return fmt.Errorf("%s %s: %w", typ, desc, err)
	}
	if typ == "expense" && amount >= 2 && created != nil {
		name := ""
		if created.Category != nil {
			name = created.Category.Name
		}
		s.expenses = append(s.expenses, expenseCand{id: created.ID, amount: amount, catName: name})
	}
	return nil
}

func floatPtrDecimal(v *float64) *decimal.Decimal {
	if v == nil {
		return nil
	}
	d := decimal.NewFromFloat(*v)
	return &d
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
