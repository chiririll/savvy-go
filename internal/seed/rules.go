package seed

import (
	"github.com/shopspring/decimal"

	"savvy-go/internal/automation"
	"savvy-go/internal/ledger"
	"savvy-go/internal/recurring"
)

func (s *seeder) createRecurring(accounts map[string]*ledger.Account, expenses, incomes []ledger.Category) error {
	if salary := catByName(incomes, "#SALARY"); salary != nil {
		day := 5
		if _, err := s.recur.Create(s.ctx, recurring.Input{
			Type: "income", AccountID: accounts["checking"].ID, CategoryID: &salary.ID,
			Amount: decimal.NewFromInt(7000), Description: strPtr("Acme Corp — Payroll"),
			Frequency: "monthly", Interval: 1, DayOfMonth: &day,
			StartDate: nextMonthOnDay(s.now, 5).Format("2006-01-02"),
		}); err != nil {
			return err
		}
	}
	if rent := catByName(expenses, "#HOUSING"); rent != nil {
		day := 1
		next := startOfMonth(s.now.AddDate(0, 1, 0)).Format("2006-01-02")
		if _, err := s.recur.Create(s.ctx, recurring.Input{
			Type: "expense", AccountID: accounts["checking"].ID, CategoryID: &rent.ID,
			Amount: decimal.NewFromInt(2150), Description: strPtr("Rent — Greystar Apartments"),
			Frequency: "monthly", Interval: 1, DayOfMonth: &day, StartDate: next,
		}); err != nil {
			return err
		}
	}
	if subs := catByName(expenses, "#SUBSCRIPTIONS"); subs != nil {
		// Always due within the coming week so upcoming recurring is visible.
		due := s.now.AddDate(0, 0, 2)
		day := due.Day()
		if _, err := s.recur.Create(s.ctx, recurring.Input{
			Type: "expense", AccountID: accounts["credit"].ID, CategoryID: &subs.ID,
			Amount: decimal.RequireFromString("22.99"), Description: strPtr("Netflix subscription"),
			Frequency: "monthly", Interval: 1, DayOfMonth: &day,
			StartDate: due.Format("2006-01-02"),
		}); err != nil {
			return err
		}
	}
	toSav := accounts["savings"].ID
	amt := decimal.NewFromInt(800)
	day6 := 6
	if _, err := s.recur.Create(s.ctx, recurring.Input{
		Type: "transfer", AccountID: accounts["checking"].ID, ToAccountID: &toSav,
		Amount: amt, ToAmount: &amt, Description: strPtr("Automatic transfer to savings"),
		Frequency: "monthly", Interval: 1, DayOfMonth: &day6,
		StartDate: nextMonthOnDay(s.now, 6).Format("2006-01-02"),
	}); err != nil {
		return err
	}
	toMort := accounts["mortgage"].ID
	mort := decimal.NewFromInt(1680)
	day2 := 2
	if _, err := s.recur.Create(s.ctx, recurring.Input{
		Type: "transfer", AccountID: accounts["checking"].ID, ToAccountID: &toMort,
		Amount: mort, ToAmount: &mort, Description: strPtr("Monthly mortgage payment"),
		Frequency: "monthly", Interval: 1, DayOfMonth: &day2,
		StartDate: nextMonthOnDay(s.now, 2).Format("2006-01-02"),
	}); err != nil {
		return err
	}
	return nil
}

func (s *seeder) createAutomation(tags []ledger.Tag) error {
	trigger := "on_transaction_create"
	if essential := tagByName(tags, "#ESSENTIAL"); essential != nil {
		desc := "Tag any transaction above $500 as Essential"
		if _, err := s.auto.Create(s.ctx, automation.Input{
			Name: "Flag large transactions", Description: &desc, TriggerType: trigger, Priority: 1,
			Conditions: map[string]any{"match": "all", "conditions": []any{
				map[string]any{"field": "amount", "op": "gt", "value": 500},
			}},
			Actions: []map[string]any{{"type": "add_tags", "tag_ids": []int64{essential.ID}}},
		}); err != nil {
			return err
		}
	}
	if recurring := tagByName(tags, "#SUBSCRIPTION"); recurring != nil {
		desc := `Tag transactions containing "subscription"`
		if _, err := s.auto.Create(s.ctx, automation.Input{
			Name: "Tag subscriptions", Description: &desc, TriggerType: trigger, Priority: 2,
			Conditions: map[string]any{"match": "all", "conditions": []any{
				map[string]any{"field": "description", "op": "contains", "value": "subscription"},
			}},
			Actions: []map[string]any{{"type": "add_tags", "tag_ids": []int64{recurring.ID}}},
		}); err != nil {
			return err
		}
	}
	if business := tagByName(tags, "#BUSINESS"); business != nil {
		desc := "Tag GitHub / Adobe / Udemy as Business"
		if _, err := s.auto.Create(s.ctx, automation.Input{
			Name: "Tag software tools", Description: &desc, TriggerType: trigger, Priority: 3,
			Conditions: map[string]any{"match": "any", "conditions": []any{
				map[string]any{"field": "description", "op": "contains", "value": "GitHub"},
				map[string]any{"field": "description", "op": "contains", "value": "Adobe"},
				map[string]any{"field": "description", "op": "contains", "value": "Udemy"},
			}},
			Actions: []map[string]any{{"type": "add_tags", "tag_ids": []int64{business.ID}}},
		}); err != nil {
			return err
		}
	}
	return nil
}
