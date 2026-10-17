package seed

import (
	"fmt"

	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
)

func (s *seeder) seedTransfersDebt(accounts map[string]*ledger.Account) error {
	for cursor := s.start; !cursor.After(s.end); cursor = addMonthsNoOverflow(cursor, 1) {
		d := dateOn(cursor, minInt(6, daysInMonth(cursor)))
		if s.inRange(d) {
			to := accounts["savings"].ID
			amt := 800.0
			if err := s.addTx("transfer", accounts["checking"].ID, nil, amt,
				"Automatic transfer to savings", d, &to, &amt, nil); err != nil {
				return err
			}
		}

		dCard := dateOn(cursor, minInt(25, daysInMonth(cursor)))
		if s.inRange(dCard) {
			pay := float64(s.mtRand(600, 1400))
			to := accounts["credit"].ID
			if err := s.addTx("debt_payment", accounts["checking"].ID, nil, pay,
				"Amex statement payment", dCard, &to, &pay, nil); err != nil {
				return err
			}
		}

		dMort := dateOn(cursor, minInt(2, daysInMonth(cursor)))
		if s.inRange(dMort) {
			to := accounts["mortgage"].ID
			amt := 1680.00
			if err := s.addTx("debt_payment", accounts["checking"].ID, nil, amt,
				"Monthly mortgage payment", dMort, &to, &amt, nil); err != nil {
				return err
			}
		}

		if accounts["eur"] != nil && int(cursor.Month())%2 == 0 {
			dFx := dateOn(cursor, minInt(14, daysInMonth(cursor)))
			if s.inRange(dFx) {
				usdOut := float64(s.mtRand(300, 600))
				toAmt := round2(usdOut / eurUSDRate)
				to := accounts["eur"].ID
				if err := s.addTx("transfer", accounts["checking"].ID, nil, usdOut,
					"USD → EUR top-up", dFx, &to, &toAmt, nil); err != nil {
					return err
				}
			}
		}
	}

	d := dateOn(s.now.AddDate(0, -2, 0), 18)
	to := accounts["loan_out"].ID
	amt := 400.0
	return s.addTx("debt_collection", accounts["checking"].ID, nil, amt,
		"Michael — partial repayment", d, &to, &amt, nil)
}

// seedPending adds one overdue pending expense and several upcoming ones (one
// with an estimated amount) so the pending strip on the transactions page has data. It does not use the rng, so
// the rest of the seeded data stays unchanged.
func (s *seeder) seedPending(accounts map[string]*ledger.Account, expenses []ledger.Category) error {
	for _, p := range []struct {
		offsetDays int
		acct, cat  string
		amount     float64
		desc       string
		estimated  bool // the amount is a guess until the real bill arrives
	}{
		{-3, "checking", "#HEALTH", 85, "Dentist appointment — City Medical Clinic", false},
		{1, "credit", "#GIFTS", 55, "Birthday gift — Etsy", false},
		{3, "checking", "#UTILITIES", 115, "ConEdison — electricity bill (estimate)", true},
		{5, "checking", "#TRANSPORT", 145, "Car insurance — GEICO", false},
		{9, "checking", "#SHOPPING", 64.50, "Amazon — scheduled order", false},
	} {
		acct := accounts[p.acct]
		if acct == nil {
			continue
		}
		var catID *int64
		if c := catByName(expenses, p.cat); c != nil {
			catID = &c.ID
		}
		d := s.now.AddDate(0, 0, p.offsetDays).Format("2006-01-02")
		desc := p.desc
		status := "pending"
		if _, err := s.txs.Create(s.ctx, ledger.TxInput{
			Type: "expense", AccountID: acct.ID, CategoryID: catID, Amount: decimal.NewFromFloat(p.amount),
			Description: &desc, Date: &d, Status: &status, IsEstimated: p.estimated,
		}); err != nil {
			return fmt.Errorf("pending %s: %w", p.desc, err)
		}
	}
	return nil
}
