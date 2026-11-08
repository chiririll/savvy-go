package ledger

import (
	"sort"

	"savvy-go/internal/money"
)

// The balance rule, defined once. A reconciliation point pins the balance at
// the end of its date: the balance at the end of day D is the latest anchor
// dated on or before D (a point's actual balance, else the opening balance
// with date "") plus the payments dated in (anchorDate, D]. Payments dated on
// the anchor's date are inside the anchor already, so the lower bound is
// exclusive. anchorOf picks the anchor; balanceLine applies the rule in memory.
// Accounts.Balance applies the same rule with SQL (one point lookup and one
// sum) because loading every day's change would cost far more for one date.

// endOfTime is a date after every real one, to look up the latest point "now".
const endOfTime = "9999-12-31"

// anchorOf is the base balance and the date after which payments count: the
// point's actual balance and date, or the opening balance and "" without one.
func anchorOf(initial money.Money, p *Reconciliation) (base money.Money, after string) {
	if p == nil {
		return initial, ""
	}
	return p.ActualBalance, p.Date
}

// dayChange is the net change of an account on one day.
type dayChange struct {
	day    string
	change money.Money
}

// balanceLine answers the balance rule in memory from an opening balance, the
// reconciliation points (ascending by date) and the daily changes (ascending
// by day, one per day).
type balanceLine struct {
	initial money.Money
	points  []Reconciliation
	changes []dayChange
	cum     []money.Money // cum[i] is the sum of changes[:i]
}

func newBalanceLine(initial money.Money, points []Reconciliation, changes []dayChange) balanceLine {
	cum := make([]money.Money, len(changes)+1)
	cum[0] = money.Zero(initial.Unit())
	for i, c := range changes {
		cum[i+1] = cum[i].Add(c.change)
	}
	return balanceLine{initial: initial, points: points, changes: changes, cum: cum}
}

// sum adds the changes dated in (after, to].
func (l balanceLine) sum(after, to string) money.Money {
	lo := sort.Search(len(l.changes), func(i int) bool { return l.changes[i].day > after })
	hi := sort.Search(len(l.changes), func(i int) bool { return l.changes[i].day > to })
	if hi < lo {
		hi = lo
	}
	return l.cum[hi].Sub(l.cum[lo])
}

// at is the balance at the end of day ("" for now).
func (l balanceLine) at(day string) money.Money {
	if day == "" {
		day = endOfTime
	}
	var latest *Reconciliation
	if i := sort.Search(len(l.points), func(i int) bool { return l.points[i].Date > day }); i > 0 {
		latest = &l.points[i-1]
	}
	base, after := anchorOf(l.initial, latest)
	return base.Add(l.sum(after, day))
}

// checks is, for each point, the balance expected at it (the previous anchor
// plus the payments in (previous anchor date, this date]) and the difference.
func (l balanceLine) checks() []ReconciliationCheck {
	out := make([]ReconciliationCheck, len(l.points))
	var prev *Reconciliation
	for i := range l.points {
		p := l.points[i]
		base, after := anchorOf(l.initial, prev)
		expected := base.Add(l.sum(after, p.Date))
		out[i] = ReconciliationCheck{Point: p, Expected: expected, Difference: p.ActualBalance.Sub(expected)}
		prev = &l.points[i]
	}
	return out
}
