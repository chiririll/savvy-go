package ledger

import (
	"testing"
)

type anchorCase struct {
	name   string
	points [][2]string // date, actual
	asOf   string
}

// All four implementations of the balance rule must agree on every case.
func TestAnchorRuleAgreesAcrossPaths(t *testing.T) {
	cases := []anchorCase{
		{"no point", nil, "2024-01-17"},
		{"no point now", nil, ""},
		{"day before point", [][2]string{{"2024-01-15", "200"}}, "2024-01-14"},
		{"on point", [][2]string{{"2024-01-15", "200"}}, "2024-01-15"},
		{"after point", [][2]string{{"2024-01-15", "200"}}, "2024-01-19"},
		{"payment after point", [][2]string{{"2024-01-15", "200"}}, "2024-01-20"},
		{"two points between", [][2]string{{"2024-01-10", "90"}, {"2024-01-20", "300"}}, "2024-01-17"},
		{"two points after", [][2]string{{"2024-01-10", "90"}, {"2024-01-20", "300"}}, "2024-02-01"},
		{"two points now", [][2]string{{"2024-01-10", "90"}, {"2024-01-20", "300"}}, ""},
		{"point date has payment", [][2]string{{"2024-01-10", "70"}}, "2024-01-10"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := pointEnv(t)
			a := e.usdAcc
			for _, p := range c.points {
				e.point(t, a, p[0], p[1])
			}
			r := (Accounts{DB: e.db}).repo()
			pts, err := r.AccountReconciliations(e.ctx, a.ID)
			if err != nil {
				t.Fatal(err)
			}
			flows, err := r.DailyFlows(e.ctx, a.ID)
			if err != nil {
				t.Fatal(err)
			}
			want := e.balanceAt(t, a, c.asOf)
			day := c.asOf
			if day == "" {
				day = endOfTime
			}
			series, err := (Accounts{DB: e.db}).BalanceSeries(e.ctx, a, []string{day})
			if err != nil {
				t.Fatal(err)
			}
			checkBal(t, "series", series[0].Decimal().String(), want)
			book := newCardBook(a.InitialBalance, flows, pts, endOfTime)
			checkBal(t, "card", book.balanceAt(c.asOf).Decimal().String(), want)
			checkBal(t, "line", book.line.at(c.asOf).Decimal().String(), want)
		})
	}
}
