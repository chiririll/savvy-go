package seed

import (
	"math"

	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
)

// seedSplitLines splits about a tenth of the demo's expenses into several
// lines of the same category, each noting what it was for.
func (s *seeder) seedSplitLines() error {
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
		lines := s.randomLines(cand.amount, catName, cand.catID)
		if len(lines) < 2 {
			continue
		}
		if err := s.txs.Modify(s.ctx, cand.id, ledger.TxInput{Lines: lines}); err != nil {
			return err
		}
	}
	return nil
}

// randomLines splits amount into two to six lines of whole cents that add up to
// it, each noted with something from the category's catalogue.
func (s *seeder) randomLines(amount float64, category string, catID *int64) []ledger.LineInput {
	catalog := catalogs[category]
	if len(catalog) == 0 {
		catalog = catalogs["#OTHER"]
	}
	remainingCents := int(math.Round(amount * 100))
	maxLines := minInt(6, minInt(len(catalog), remainingCents))
	if maxLines < 2 {
		return nil
	}
	names := append([]string{}, catalog...)
	s.rng.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })
	names = names[:s.mtRand(2, maxLines)]

	var lines []ledger.LineInput
	for i := range names {
		left := len(names) - i - 1
		cents := remainingCents
		if left > 0 {
			maxShare := remainingCents - left
			cents = maxInt(1, (maxShare*s.mtRand(25, 55))/100)
		}
		remainingCents -= cents
		note := names[i]
		lines = append(lines, ledger.LineInput{CategoryID: catID, Amount: decimal.New(int64(cents), -2), Note: &note})
	}
	return lines
}
