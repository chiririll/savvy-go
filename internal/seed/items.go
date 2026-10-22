package seed

import (
	"math"

	"github.com/shopspring/decimal"

	ledgerrepo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
)

func (s *seeder) seedTransactionItems() error {
	if len(s.expenses) == 0 {
		return nil
	}
	target := int(math.Round(float64(s.countTx()) * 0.10))
	if target < 1 {
		target = 1
	}
	s.rng.Shuffle(len(s.expenses), func(i, j int) {
		s.expenses[i], s.expenses[j] = s.expenses[j], s.expenses[i]
	})
	if target > len(s.expenses) {
		target = len(s.expenses)
	}
	for _, cand := range s.expenses[:target] {
		catName := cand.catName
		if catName == "" {
			catName = "#OTHER"
		}
		items := s.randomItems(cand.amount, catName)
		cents := money.Unit{Decimals: 2} // the demo prices items in cents
		lines := make([]ledgerrepo.NewTxItem, len(items))
		for i, it := range items {
			lines[i] = ledgerrepo.NewTxItem{
				Name: it.Name, Quantity: it.Quantity,
				PricePerUnit: money.FromDecimal(it.PricePerUnit, cents), TotalPrice: money.FromDecimal(it.TotalPrice, cents),
			}
		}
		if err := (ledgerrepo.Repo{DB: s.db}).InsertItems(s.ctx, cand.id, lines, s.now.UTC()); err != nil {
			return err
		}
	}
	return nil
}

func (s *seeder) randomItems(amount float64, category string) []ledger.TxItemInput {
	catalog := catalogs[category]
	if len(catalog) == 0 {
		catalog = catalogs["#OTHER"]
	}
	remainingCents := int(math.Round(amount * 100))
	if remainingCents < 1 {
		return nil
	}
	maxItems := minInt(6, minInt(len(catalog), remainingCents))
	count := 1
	if maxItems > 1 {
		count = s.mtRand(2, maxItems)
	}
	names := append([]string{}, catalog...)
	s.rng.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })
	names = names[:count]

	var items []ledger.TxItemInput
	left := len(names)
	for _, name := range names {
		left--
		canSplit := left > 0 && remainingCents > left
		if !canSplit {
			if remainingCents < 1 {
				break
			}
			price := decimal.New(int64(remainingCents), -2)
			items = append(
				items,
				ledger.TxItemInput{Name: name, Quantity: decimal.NewFromInt(1), PricePerUnit: price, TotalPrice: price},
			)
			break
		}
		maxShare := remainingCents - left
		qty := s.mtRand(1, minInt(3, maxShare))
		share := maxInt(qty, (maxShare*s.mtRand(25, 55))/100)
		lineCents := (share / qty) * qty
		lineCents = maxInt(qty, minInt(lineCents, (maxShare/qty)*qty))
		remainingCents -= lineCents
		items = append(items, ledger.TxItemInput{
			Name: name, Quantity: decimal.NewFromInt(int64(qty)),
			PricePerUnit: decimal.New(int64(lineCents/qty), -2),
			TotalPrice:   decimal.New(int64(lineCents), -2),
		})
	}
	return items
}
