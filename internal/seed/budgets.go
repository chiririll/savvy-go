package seed

import (
	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
)

func (s *seeder) createBudgets(usd *ledger.Currency, expenses []ledger.Category, tags []ledger.Tag) error {
	start := startOfMonth(s.now).Format("2006-01-02")
	type named struct {
		name   string
		amount float64
		cat    string
		notify int
	}
	for _, b := range []named{
		{"Groceries", 750, "#GROCERIES", 80},
		{"Dining Out", 450, "#DINING", 90},
		{"Fun Money", 300, "#ENTERTAINMENT", 100},
	} {
		cat := catByName(expenses, b.cat)
		if cat == nil {
			continue
		}
		notify := b.notify
		if _, err := s.budgets.Create(s.ctx, ledger.BudgetInput{
			Name: b.name, Amount: decimal.NewFromFloat(b.amount), CurrencyID: &usd.ID, Period: "monthly",
			StartDate: &start, NotifyAtPercent: &notify, CategoryIDs: []int64{cat.ID},
		}); err != nil {
			return err
		}
	}
	notify := 85
	global := true
	if _, err := s.budgets.Create(s.ctx, ledger.BudgetInput{
		Name: "Total Monthly Spending", Amount: decimal.NewFromInt(5500), CurrencyID: &usd.ID, Period: "monthly",
		StartDate: &start, IsGlobal: &global, NotifyAtPercent: &notify,
	}); err != nil {
		return err
	}
	if vac := tagByName(tags, "Vacation"); vac != nil {
		from := s.now.AddDate(0, -1, 0).Format("2006-01-02")
		to := s.now.AddDate(0, 4, 0).Format("2006-01-02")
		n := 75
		if _, err := s.budgets.Create(s.ctx, ledger.BudgetInput{
			Name: "Italy Trip 2026", Amount: decimal.NewFromInt(3500), CurrencyID: &usd.ID, Period: "one_time",
			StartDate: &from, EndDate: &to, NotifyAtPercent: &n, TagIDs: []int64{vac.ID},
		}); err != nil {
			return err
		}
	}
	return nil
}
