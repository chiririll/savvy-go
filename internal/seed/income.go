package seed

import (
	"fmt"
	"time"

	"savvy-go/internal/ledger"
)

func (s *seeder) seedIncome(accounts map[string]*ledger.Account, incomes []ledger.Category) error {
	salary := catByName(incomes, "#SALARY")
	freelance := catByName(incomes, "#SIDE_INCOME")
	investments := catByName(incomes, "#INVESTMENTS")
	other := catByName(incomes, "#OTHER_INCOME")

	// The random draws of each month happen in this order; they are part of the demo data.
	for cursor := s.start; !cursor.After(s.end); cursor = addMonthsNoOverflow(cursor, 1) {
		if err := s.seedSalary(accounts, salary, cursor); err != nil {
			return err
		}
		if err := s.seedFreelance(accounts, freelance, cursor); err != nil {
			return err
		}
		if err := s.seedDividend(accounts, investments, cursor); err != nil {
			return err
		}
	}

	if other != nil {
		d := dateOn(s.now.AddDate(0, -2, 0), 12)
		if err := s.addTx("income", accounts["checking"].ID, &other.ID,
			float64(s.mtRand(900, 1400)), "Tax refund — IRS", d, nil, nil, nil); err != nil {
			return err
		}
	}
	return nil
}

// seedSalary adds the two paydays of the month of cursor.
func (s *seeder) seedSalary(accounts map[string]*ledger.Account, salary *ledger.Category, cursor time.Time) error {
	const baseSalary = 6400.0
	for _, payDay := range []int{5, 20} {
		payday := dateOn(cursor, minInt(payDay, daysInMonth(cursor)))
		if salary == nil || !s.inRange(payday) {
			continue
		}
		raise := 0.0
		if monthsBetween(s.start, cursor) >= 7 {
			raise = 600
		}
		if err := s.addTx("income", accounts["checking"].ID, &salary.ID,
			baseSalary+raise+float64(s.mtRand(-40, 40)),
			"Acme Corp — Payroll", payday, nil, nil, nil); err != nil {
			return err
		}
	}
	return nil
}

// seedFreelance adds a freelance payout to most months.
func (s *seeder) seedFreelance(accounts map[string]*ledger.Account, freelance *ledger.Category,
	cursor time.Time) error {
	if freelance == nil || s.mtRand(1, 100) > 65 {
		return nil
	}
	d := dateOn(cursor, s.mtRand(8, minInt(26, daysInMonth(cursor))))
	if !s.inRange(d) {
		return nil
	}
	vendors := []string{"Upwork payout", "Fiverr withdrawal", fmt.Sprintf("Client invoice #%d", s.mtRand(1000, 9999))}
	acct := accounts["checking"]
	if accounts["eur"] != nil && s.mtRand(0, 1) == 1 {
		acct = accounts["eur"]
	}
	return s.addTx("income", acct.ID, &freelance.ID, float64(s.mtRand(450, 1900)),
		vendors[s.mtRand(0, 2)], d, nil, nil, nil)
}

// seedDividend adds the quarterly dividend in the first month of each quarter.
func (s *seeder) seedDividend(accounts map[string]*ledger.Account, investments *ledger.Category,
	cursor time.Time) error {
	if investments == nil || int(cursor.Month())%3 != 1 {
		return nil
	}
	d := dateOn(cursor, minInt(15, daysInMonth(cursor)))
	if !s.inRange(d) {
		return nil
	}
	return s.addTx("income", accounts["crypto"].ID, &investments.ID,
		float64(s.mtRand(80, 420)), "Quarterly dividend payout", d, nil, nil, nil)
}
