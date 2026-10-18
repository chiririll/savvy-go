package reports

import (
	"context"
	"sort"
	"time"

	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
)

type nwAccount struct {
	ID      int64
	Name    string
	Type    string
	Balance money.Money // in the base currency
}

// accounts are the active, non-debt accounts the report covers.
func (q report) accounts() []ledger.Account {
	list, err := ledger.Accounts{DB: q.DB}.All(q.ctx, true, true)
	if err != nil || q.base == nil {
		return nil
	}
	if len(q.f.AccountIDs) == 0 {
		return list
	}
	want := map[int64]bool{}
	for _, id := range q.f.AccountIDs {
		want[id] = true
	}
	out := list[:0]
	for _, a := range list {
		if want[a.ID] {
			out = append(out, a)
		}
	}
	return out
}

// netWorthAt is each account's balance at the end of day at, converted to the
// base currency.
func (q report) netWorthAt(accts []ledger.Account, at time.Time) []nwAccount {
	cutoff := at.Format("2006-01-02")
	out := make([]nwAccount, 0, len(accts))
	for _, a := range accts {
		bal, _ := ledger.Accounts{DB: q.DB}.Balance(q.ctx, a, cutoff)
		out = append(out, nwAccount{ID: a.ID, Name: a.Name, Type: a.Type, Balance: ledger.Convert(bal, *a.Currency, *q.base)})
	}
	return out
}

func (q report) totalOf(list []nwAccount) money.Money {
	total := q.zero()
	for _, a := range list {
		total = total.Add(a.Balance)
	}
	return total
}

func (s Service) NetWorth(ctx context.Context, f Filter) NetWorthReport {
	q := s.open(ctx, f)
	now := s.now()
	accts := q.accounts()
	current := q.netWorthAt(accts, f.Range(now).End)
	curTotal := q.totalOf(current)
	var previous *money.Money
	change := q.zero()
	changePct := 0.0
	if cmp := f.Comparison(now); cmp != nil {
		prevTotal := q.totalOf(q.netWorthAt(accts, cmp.End))
		previous = &prevTotal
		change = curTotal.Sub(prevTotal)
		if !prevTotal.IsZero() {
			changePct = percent(change, prevTotal.Abs())
		}
	}
	sort.SliceStable(current, func(i, j int) bool { return current[i].Balance.Cmp(current[j].Balance) > 0 })
	rows := []NetWorthAccount{}
	for _, a := range current {
		pct := 0.0
		if curTotal.IsPositive() {
			pct = percent(a.Balance, curTotal)
		}
		rows = append(rows, NetWorthAccount{ID: a.ID, Name: a.Name, Type: a.Type, Balance: a.Balance, Percentage: pct})
	}
	return NetWorthReport{
		Current: curTotal, Previous: previous,
		Change: change, ChangePercent: changePct,
		Accounts: rows, Currency: q.code(),
	}
}

func (s Service) NetWorthHistory(ctx context.Context, f Filter, groupBy string) NetWorthHistoryReport {
	q := s.open(ctx, f)
	r := f.Range(s.now())
	accts := q.accounts()
	dates, values := []string{}, []money.Money{}
	for cur := r.Start; !cur.After(r.End); cur = nextPeriod(cur, groupBy) {
		dates = append(dates, cur.Format("2006-01-02"))
		values = append(values, q.totalOf(q.netWorthAt(accts, periodEnd(cur, r.End, groupBy))))
	}
	return NetWorthHistoryReport{Dates: dates, Values: values, Currency: q.code()}
}
