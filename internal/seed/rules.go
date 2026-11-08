package seed

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/automation"
	"savvy-go/internal/ledger"
	"savvy-go/internal/recurring"
)

// rule makes a monthly recurring rule from the transaction it creates. Its
// template is the JSON of a transaction input, as the API takes it.
func (s *seeder) rule(in ledger.TxInput, day int, start time.Time) error {
	template, err := templateJSON(in)
	if err != nil {
		return err
	}
	_, err = s.recur.Create(s.ctx, recurring.Input{
		Template: template,
		Schedule: recurring.Schedule{Frequency: "monthly", Interval: 1, DayOfMonth: &day, Start: start.Format("2006-01-02")},
	})
	return err
}

// templateJSON is a template as stored. Lists the input leaves out are empty,
// not null: clients read the stored JSON as it is.
func templateJSON(in ledger.TxInput) ([]byte, error) {
	if in.Payments == nil {
		in.Payments = []ledger.PaymentInput{}
	}
	if in.Lines == nil {
		in.Lines = []ledger.LineInput{}
	}
	if in.TagIDs == nil {
		in.TagIDs = []int64{}
	}
	return json.Marshal(in)
}

// flow is an income or an expense of amount on an account, as a template.
func flow(typ string, account int64, category *int64, amount, desc string) ledger.TxInput {
	line := decimal.RequireFromString(amount)
	paid := line
	if typ == "expense" {
		paid = line.Neg()
	}
	return ledger.TxInput{
		Type: typ, Description: strPtr(desc),
		Payments: []ledger.PaymentInput{{AccountID: account, Amount: &paid}},
		Lines:    []ledger.LineInput{{CategoryID: category, Amount: line}},
	}
}

func (s *seeder) createRecurring(accounts map[string]*ledger.Account, expenses, incomes []ledger.Category) error {
	if salary := catByName(incomes, "#SALARY"); salary != nil {
		in := flow("income", accounts["checking"].ID, &salary.ID, "7000", "Acme Corp — Payroll")
		if err := s.rule(in, 5, nextMonthOnDay(s.now, 5)); err != nil {
			return err
		}
	}
	if rent := catByName(expenses, "#HOUSING"); rent != nil {
		in := flow("expense", accounts["checking"].ID, &rent.ID, "2150", "Rent — Greystar Apartments")
		if err := s.rule(in, 1, startOfMonth(s.now.AddDate(0, 1, 0))); err != nil {
			return err
		}
	}
	if subs := catByName(expenses, "#SUBSCRIPTIONS"); subs != nil {
		// Always due within the coming week so upcoming recurring is visible.
		due := s.now.AddDate(0, 0, 2)
		in := flow("expense", accounts["credit"].ID, &subs.ID, "22.99", "Netflix subscription")
		if err := s.rule(in, due.Day(), due); err != nil {
			return err
		}
	}
	// The receiving side of the transfer has no amount: each occurrence takes
	// what balances the other side at that day's rates.
	out := decimal.NewFromInt(-800)
	save := ledger.TxInput{
		Type: "transfer", Description: strPtr("Automatic transfer to savings"),
		Payments: []ledger.PaymentInput{
			{AccountID: accounts["checking"].ID, Amount: &out},
			{AccountID: accounts["savings"].ID, Estimated: true},
		},
	}
	// The mortgage payment is not repeated here: it is a debt transaction, and a
	// rule creates only what can be entered by hand.
	return s.rule(save, 6, nextMonthOnDay(s.now, 6))
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
