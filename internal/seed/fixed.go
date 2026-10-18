package seed

import (
	"time"

	"savvy-go/internal/ledger"
)

// fixedSubs are the monthly subscriptions, charged to the credit card.
var fixedSubs = []struct {
	name  string
	price float64
}{
	{"Netflix", 22.99}, {"Spotify", 11.99}, {"iCloud+", 9.99},
	{"Adobe Creative Cloud", 59.99}, {"ChatGPT Plus", 20.00}, {"GitHub", 4.00},
}

// fixedUtilities are the monthly bills, with the range of their amounts.
var fixedUtilities = []struct {
	name   string
	lo, hi int
}{
	{"ConEdison", 70, 180},
	{"Xfinity Internet", 79, 79},
	{"National Grid Gas", 40, 120},
}

func (s *seeder) seedFixedExpenses(accounts map[string]*ledger.Account, expenses []ledger.Category) error {
	rent := catByName(expenses, "#HOUSING")
	utilities := catByName(expenses, "#UTILITIES")
	entertainment := catByName(expenses, "#SUBSCRIPTIONS")
	for cursor := s.start; !cursor.After(s.end); cursor = addMonthsNoOverflow(cursor, 1) {
		if rent != nil {
			if err := s.seedRent(accounts, rent, cursor); err != nil {
				return err
			}
		}
		if utilities != nil {
			if err := s.seedUtilities(accounts, utilities, cursor); err != nil {
				return err
			}
		}
		if entertainment != nil {
			if err := s.seedSubscriptions(accounts, entertainment, cursor); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *seeder) seedRent(accounts map[string]*ledger.Account, rent *ledger.Category, month time.Time) error {
	d := startOfMonth(month)
	if !s.inRange(d) {
		return nil
	}
	return s.addTx("expense", accounts["checking"].ID, &rent.ID, 2150,
		"Rent — Greystar Apartments", d, nil, nil, nil)
}

func (s *seeder) seedUtilities(accounts map[string]*ledger.Account, cat *ledger.Category, month time.Time) error {
	for _, u := range fixedUtilities {
		d := dateOn(month, minInt(s.mtRand(12, 18), daysInMonth(month)))
		if !s.inRange(d) {
			continue
		}
		winter := 1.0
		if m := int(month.Month()); m == 12 || m == 1 || m == 2 {
			winter = 1.4
		}
		amt := round2(float64(s.mtRand(u.lo, u.hi)) * winter)
		if err := s.addTx("expense", accounts["checking"].ID, &cat.ID, amt, u.name, d, nil, nil, nil); err != nil {
			return err
		}
	}
	return nil
}

func (s *seeder) seedSubscriptions(accounts map[string]*ledger.Account, cat *ledger.Category, month time.Time) error {
	for _, sub := range fixedSubs {
		d := dateOn(month, minInt(s.mtRand(2, 9), daysInMonth(month)))
		if !s.inRange(d) {
			continue
		}
		if err := s.addTx("expense", accounts["credit"].ID, &cat.ID, sub.price,
			sub.name+" subscription", d, nil, nil, nil); err != nil {
			return err
		}
	}
	return nil
}
