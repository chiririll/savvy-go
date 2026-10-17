package seed

import (
	"math"
	"time"

	"savvy-go/internal/ledger"
)

// variableWeights is how often each variable expense category is drawn.
var variableWeights = []struct {
	name string
	w    int
}{
	{"#GROCERIES", 22}, {"#DINING", 20}, {"#TRANSPORT", 16},
	{"#SHOPPING", 12}, {"#ENTERTAINMENT", 9}, {"#HEALTH", 6}, {"#PERSONAL_CARE", 5},
	{"#TRAVEL", 3}, {"#HOUSING", 2}, {"#GIFTS", 2}, {"#OTHER", 2},
}

// variableRanges is the whole-unit amount range per variable expense category.
var variableRanges = map[string][2]int{
	"#GROCERIES": {18, 145}, "#DINING": {9, 70}, "#TRANSPORT": {4, 65},
	"#SHOPPING": {15, 240}, "#ENTERTAINMENT": {12, 95}, "#HEALTH": {10, 160},
	"#PERSONAL_CARE": {15, 85}, "#TRAVEL": {120, 950}, "#HOUSING": {25, 180},
	"#GIFTS": {20, 180}, "#OTHER": {10, 120},
}

// variableTags are the tags the variable expenses may carry.
type variableTags struct{ vacation, essential, business *ledger.Tag }

func (s *seeder) seedVariableExpenses(
	accounts map[string]*ledger.Account,
	expenses []ledger.Category,
	tags []ledger.Tag,
) error {
	var pool []ledger.Category
	for _, w := range variableWeights {
		if c := catByName(expenses, w.name); c != nil {
			for i := 0; i < w.w; i++ {
				pool = append(pool, *c)
			}
		}
	}
	if len(pool) == 0 {
		return nil
	}
	vt := variableTags{tagByName(tags, "Vacation"), tagByName(tags, "Essential"), tagByName(tags, "Business")}

	for cursor := s.start; !cursor.After(s.end); cursor = cursor.AddDate(0, 0, 1) {
		seasonal := seasonalFactor(cursor.Month())
		lo, hi := 1, 3
		if cursor.Weekday() == time.Saturday || cursor.Weekday() == time.Sunday {
			lo, hi = 2, 5
		}
		count := int(math.Round(float64(s.mtRand(lo, hi)) * seasonal))
		for i := 0; i < count; i++ {
			cat := pool[s.rng.Intn(len(pool))]
			if err := s.seedVariableExpense(accounts, vt, cat, cursor, seasonal); err != nil {
				return err
			}
		}
	}
	return nil
}

func seasonalFactor(m time.Month) float64 {
	switch m {
	case time.November, time.December:
		return 1.5
	case time.July:
		return 1.25
	}
	return 1.0
}

// seedVariableExpense adds one purchase in cat on day. The order of the random
// draws is part of the demo data, so keep it.
func (s *seeder) seedVariableExpense(
	accounts map[string]*ledger.Account,
	vt variableTags,
	cat ledger.Category,
	day time.Time,
	seasonal float64,
) error {
	if cat.Name == "#TRAVEL" && s.mtRand(1, 100) > 12 {
		return nil
	}
	r := variableRanges[cat.Name]
	if r[1] == 0 {
		r = [2]int{10, 80}
	}
	amount := round2(float64(s.mtRand(r[0]*100, r[1]*100)) / 100 * seasonal)
	merchant := s.merchantFor(cat.Name)
	acct := s.variableAccount(accounts, cat.Name, amount)
	tagIDs := s.variableTagIDs(vt, cat.Name)
	return s.addTx("expense", acct.ID, &cat.ID, amount, merchant, day, nil, nil, tagIDs)
}

func (s *seeder) variableAccount(accounts map[string]*ledger.Account, cat string, amount float64) *ledger.Account {
	switch {
	case cat == "#TRAVEL" && accounts["eur"] != nil && s.mtRand(0, 1) == 1:
		return accounts["eur"]
	case inList(cat, "#GROCERIES", "#SHOPPING", "#TRAVEL", "#HEALTH") && s.mtRand(1, 100) <= 60:
		return accounts["credit"]
	case amount < 25 && s.mtRand(1, 100) <= 35:
		return accounts["cash"]
	}
	return accounts["checking"]
}

func (s *seeder) variableTagIDs(vt variableTags, cat string) []int64 {
	var ids []int64
	if cat == "#TRAVEL" && vt.vacation != nil {
		ids = append(ids, vt.vacation.ID)
	}
	if inList(cat, "#GROCERIES", "#HEALTH") && vt.essential != nil && s.mtRand(1, 100) <= 35 {
		ids = append(ids, vt.essential.ID)
	}
	if cat == "#HOUSING" && vt.business != nil && s.mtRand(1, 100) <= 20 {
		ids = append(ids, vt.business.ID)
	}
	return ids
}
