package seed

import (
	"fmt"
	"time"

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
	leg := func(typ string, account *ledger.Account, amount string, desc string, day time.Time) ledger.TxInput {
		d := day.Format("2006-01-02")
		return ledger.TxInput{
			Type: typ, AccountID: account.ID, Amount: decimal.RequireFromString(amount),
			Description: &desc, Date: &d,
		}
	}
	shop := leg("expense", accounts["cash"], "18.50", "Farmers market — cash and card", s.now.AddDate(0, 0, -3))
	shop.CategoryID = &groceries.ID
	card := leg("expense", accounts["checking"], "41.50", "Farmers market — cash and card", s.now.AddDate(0, 0, -3))
	card.CategoryID = &groceries.ID
	if _, err := (ledger.Groups{DB: s.db}).Create(s.ctx, []ledger.TxInput{shop, card}); err != nil {
		return fmt.Errorf("split groceries: %w", err)
	}

	// 40 EUR for souvenirs, paid with 50 EUR; the 10 EUR left came back as 11.50 USD.
	day := s.now.AddDate(0, 0, -9)
	souvenirs := leg("expense", accounts["eur"], "40", "Souvenirs — paid in euros, change in dollars", day)
	souvenirs.CategoryID = &travel.ID
	exchange := leg("transfer", accounts["eur"], "10", "Souvenirs — paid in euros, change in dollars", day)
	back := decimal.RequireFromString("11.50")
	exchange.ToAccountID, exchange.ToAmount = &accounts["cash"].ID, &back
	if _, err := (ledger.Groups{DB: s.db}).Create(s.ctx, []ledger.TxInput{souvenirs, exchange}); err != nil {
		return fmt.Errorf("split souvenirs: %w", err)
	}
	return nil
}
