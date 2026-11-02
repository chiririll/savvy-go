package seed

import (
	"fmt"

	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
)

// seedSplitPayments adds two payments made in parts, so the demo shows them:
// a grocery bill paid with cash and card, and a purchase in euros paid with a
// larger note, the change coming back in dollars.
func (s *seeder) seedSplitPayments(accounts map[string]*ledger.Account, expenses []ledger.Category) error {
	groceries := catByName(expenses, "#GROCERIES")
	travel := catByName(expenses, "#TRAVEL")
	if groceries == nil || travel == nil || accounts["eur"] == nil {
		return nil
	}
	dec := decimal.RequireFromString
	day := func(offset int) *string {
		d := s.now.AddDate(0, 0, offset).Format("2006-01-02")
		return &d
	}
	cash, card := dec("-18.50"), dec("-41.50")
	desc := "Farmers market — cash and card"
	if _, err := s.txs.Create(s.ctx, ledger.TxInput{
		Type: "expense", Description: &desc, Date: day(-3),
		Payments: []ledger.PaymentInput{
			{AccountID: accounts["cash"].ID, Amount: &cash}, {AccountID: accounts["checking"].ID, Amount: &card},
		},
		Lines: []ledger.LineInput{{CategoryID: &groceries.ID, Amount: dec("60")}},
	}); err != nil {
		return fmt.Errorf("split groceries: %w", err)
	}

	// 40 EUR for souvenirs, paid with 50 EUR; the 10 EUR left came back as 11.50 USD.
	paid, change := dec("-50"), dec("11.50")
	desc = "Souvenirs — paid in euros, change in dollars"
	if _, err := s.txs.Create(s.ctx, ledger.TxInput{
		Type: "expense", Description: &desc, Date: day(-9),
		Payments: []ledger.PaymentInput{
			{AccountID: accounts["eur"].ID, Amount: &paid}, {AccountID: accounts["cash"].ID, Amount: &change},
		},
		Lines: []ledger.LineInput{{CategoryID: &travel.ID, CurrencyID: accounts["eur"].CurrencyID, Amount: dec("40")}},
	}); err != nil {
		return fmt.Errorf("split souvenirs: %w", err)
	}
	return nil
}
