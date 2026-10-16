package reports

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db/filter"
	ledgerrepo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
	"savvy-go/internal/platform/calendar"
)

// Every amount a report returns is a money.Money in the base currency. Sums
// over several currencies are folded and rounded once, in internal/db/filter;
// what is computed here (balances, differences, averages, shares) stays in
// Money, so nothing is rounded twice or leaves the base scale. Percentages are
// not money and stay float64.

// report is one report request: the filter plus the base currency every amount
// in the answer is expressed in. It is loaded once per request so the helpers
// below do not each look the currency up again.
type report struct {
	Service
	ctx  context.Context
	f    Filter
	base *ledger.Currency // nil when there is no currency yet
	unit money.Unit       // the unit of base; the zero Unit without one
}

func (s Service) open(ctx context.Context, f Filter) report {
	q := report{Service: s, ctx: ctx, f: f}
	if base, _ := (ledger.Currencies{DB: s.DB}).Base(ctx); base != nil {
		q.base, q.unit = base, base.Unit()
	}
	return q
}

func (q report) zero() money.Money { return money.Zero(q.unit) }

// code is the base currency code; empty without a base.
func (q report) code() string {
	if q.base == nil {
		return ""
	}
	return q.base.Code
}

func (q report) where(typ string, r dateRange, categoryID int64) filter.ReportWhere {
	return filter.ReportWhere{
		Type: typ, Start: r.Start.Format("2006-01-02"), End: r.End.Format("2006-01-02"),
		CategoryID: categoryID, AccountIDs: q.f.AccountIDs, CategoryIDs: q.f.CategoryIDs, TagIDs: q.f.TagIDs,
	}
}

// lookup answers a stored value, or none for a key with no data: a Money
// missing from a map would otherwise come back without a unit.
type lookup[K comparable, V any] struct {
	m    map[K]V
	none V
}

func (l lookup[K, V]) at(key K) V {
	if v, ok := l.m[key]; ok {
		return v
	}
	return l.none
}

// percent is part as a percentage of whole, to one decimal. whole must not be
// zero. Both are in the same unit, so minor units give the ratio directly.
func percent(part, whole money.Money) float64 {
	return math.Round(float64(part.Minor())/float64(whole.Minor())*1000) / 10
}

// portion is total * part / whole, rounded into the base unit.
func (q report) portion(total, part, whole money.Money) money.Money {
	d := total.Decimal().Mul(part.Decimal()).DivRound(whole.Decimal(), money.RateDivPrecision)
	return money.FromDecimal(d, q.unit)
}

// average is total spread over days, scaled to per days (1 for a day, 7 for a
// week).
func (q report) average(total money.Money, days, per int) money.Money {
	d := total.Decimal().Mul(decimal.NewFromInt(int64(per))).DivRound(decimal.NewFromInt(int64(days)), money.RateDivPrecision)
	return money.FromDecimal(d, q.unit)
}

func (q report) sum(typ string, r dateRange, categoryID int64) money.Money {
	return filter.SumByType(q.ctx, q.DB, q.where(typ, r, categoryID), q.unit)
}

type catTotal struct {
	ID    int64
	Name  string
	Icon  string
	Color string
	Total money.Money
	Node  string
}

func (q report) byCategory(typ string, r dateRange) []catTotal {
	var out []catTotal
	for _, x := range filter.SumGroupedByCategory(q.ctx, q.DB, q.where(typ, r, 0), q.unit) {
		out = append(out, catTotal{
			ID: x.ID, Name: x.Name, Icon: coalesce(x.Icon.String, "circle"),
			Color: coalesce(x.Color.String, "#64748b"), Total: x.Total,
		})
	}
	return out
}

type dayTotal struct {
	Total money.Money
	Count int
}

func (q report) daily(typ string, r dateRange) lookup[string, dayTotal] {
	out := lookup[string, dayTotal]{m: map[string]dayTotal{}, none: dayTotal{Total: q.zero()}}
	for _, x := range filter.DailyTotals(q.ctx, q.DB, q.where(typ, r, 0), q.unit) {
		out.m[x.Day] = dayTotal{Total: x.Total, Count: x.Count}
	}
	return out
}

func (q report) byPeriod(typ string, r dateRange, groupBy string, categoryID int64) lookup[string, money.Money] {
	out := lookup[string, money.Money]{m: map[string]money.Money{}, none: q.zero()}
	for _, x := range filter.GroupedByPeriod(q.ctx, q.DB, q.where(typ, r, categoryID), groupBy, q.unit) {
		out.m[x.Key] = x.Total
	}
	return out
}

func (s Service) Overview(ctx context.Context, f Filter) OverviewReport {
	q := s.open(ctx, f)
	now := s.now()
	cur := q.metrics(f.Range(now))
	var prev *reportMetrics
	if cmp := f.Comparison(now); cmp != nil {
		m := q.metrics(*cmp)
		prev = &m
	}
	var incSpark, expSpark, netSpark []money.Money
	var savSpark []float64
	for _, p := range f.Sparkline(now, 6) {
		m := q.metrics(p)
		incSpark = append(incSpark, m.Income)
		expSpark = append(expSpark, m.Expenses)
		netSpark = append(netSpark, m.Net)
		savSpark = append(savSpark, m.SavingsRate)
	}
	return OverviewReport{
		Income:      Metric[money.Money]{cur.Income, prevMetric(prev, func(m reportMetrics) money.Money { return m.Income }), incSpark},
		Expenses:    Metric[money.Money]{cur.Expenses, prevMetric(prev, func(m reportMetrics) money.Money { return m.Expenses }), expSpark},
		NetCashFlow: Metric[money.Money]{cur.Net, prevMetric(prev, func(m reportMetrics) money.Money { return m.Net }), netSpark},
		SavingsRate: Metric[float64]{cur.SavingsRate, prevMetric(prev, func(m reportMetrics) float64 { return m.SavingsRate }), savSpark},
		Currency:    q.code(),
	}
}

type reportMetrics struct {
	Income, Expenses, Net money.Money
	SavingsRate           float64
}

func (q report) metrics(r dateRange) reportMetrics {
	income, expenses := q.sum("income", r, 0), q.sum("expense", r, 0)
	net := income.Sub(expenses)
	rate := 0.0
	switch {
	case income.IsPositive():
		rate = percent(net, income)
	case expenses.IsPositive():
		rate = -100
	}
	return reportMetrics{Income: income, Expenses: expenses, Net: net, SavingsRate: rate}
}

// prevMetric is a metric of the comparison period; nil without one.
func prevMetric[T any](p *reportMetrics, get func(reportMetrics) T) *T {
	if p == nil {
		return nil
	}
	return ptrTo(get(*p))
}

func (s Service) MoneyFlow(ctx context.Context, f Filter) MoneyFlowReport {
	q := s.open(ctx, f)
	r := f.Range(s.now())
	income, expenses := q.byCategory("income", r), q.byCategory("expense", r)
	used := map[string]bool{}
	resolve := func(name string) string {
		c := name
		for used[c] {
			c += "​"
		}
		used[c] = true
		return c
	}
	for i := range expenses {
		expenses[i].Node = resolve(expenses[i].Name)
	}
	for i := range income {
		income[i].Node = resolve(income[i].Name)
	}
	savingsNode := resolve("__savings__")
	totalInc, totalExp := q.zero(), q.zero()
	for _, x := range income {
		totalInc = totalInc.Add(x.Total)
	}
	for _, x := range expenses {
		totalExp = totalExp.Add(x.Total)
	}
	savings := totalInc.Sub(totalExp)
	incomeColors := []string{"#22c55e", "#16a34a", "#15803d", "#14532d", "#166534", "#4ade80"}
	expenseColors := []string{"#ef4444", "#f97316", "#eab308", "#ec4899", "#8b5cf6", "#06b6d4", "#f43f5e", "#a855f7"}
	nodes := []FlowNode{}
	for i, x := range income {
		nodes = append(nodes, FlowNode{Name: x.Node, Color: incomeColors[i%len(incomeColors)]})
	}
	for i, x := range expenses {
		nodes = append(nodes, FlowNode{Name: x.Node, Color: expenseColors[i%len(expenseColors)]})
	}
	if savings.IsPositive() {
		nodes = append(nodes, FlowNode{Name: savingsNode, Color: "#3b82f6"})
	}
	links := []FlowLink{}
	if totalInc.IsPositive() && totalExp.IsPositive() {
		for _, inc := range income {
			for _, exp := range expenses {
				if v := q.portion(exp.Total, inc.Total, totalInc); v.IsPositive() {
					links = append(links, FlowLink{Source: inc.Node, Target: exp.Node, Value: v})
				}
			}
			if savings.IsPositive() {
				if v := q.portion(savings, inc.Total, totalInc); v.IsPositive() {
					links = append(links, FlowLink{Source: inc.Node, Target: savingsNode, Value: v})
				}
			}
		}
	}
	return MoneyFlowReport{
		Nodes: nodes, Links: links,
		Totals:   FlowTotals{Income: totalInc, Expenses: totalExp, Savings: money.Max(q.zero(), savings)},
		Currency: q.code(),
	}
}

func (s Service) ExpensePace(ctx context.Context, f Filter) ExpensePaceReport {
	q := s.open(ctx, f)
	now := s.now()
	r := f.Range(now)
	daily := q.daily("expense", r)
	budget := q.monthlyBudget()
	today := dateOnly(now)
	months := []PaceMonth{}
	cursor := calendar.StartOfMonth(r.Start)
	for !cursor.After(r.End) {
		if cursor.After(today) && (cursor.Year() != today.Year() || cursor.Month() != today.Month()) {
			break
		}
		days := calendar.DaysInMonth(cursor)
		cumulative := make([]money.Money, 0, days)
		total := q.zero()
		for i := 0; i < days; i++ {
			day := cursor.AddDate(0, 0, i)
			if !day.Before(r.Start) && !day.After(r.End) {
				total = total.Add(daily.at(day.Format("2006-01-02")).Total)
			}
			cumulative = append(cumulative, total)
		}
		var currentDay *int
		if cursor.Year() == today.Year() && cursor.Month() == today.Month() {
			currentDay = ptrTo(today.Day())
		}
		months = append(months, PaceMonth{
			Budget: budget, DailyExpenses: cumulative,
			CurrentDay: currentDay, DaysInMonth: days, TotalSpent: total,
			MonthStart: cursor.Format("2006-01-02"), MonthEnd: calendar.EndOfMonth(cursor).Format("2006-01-02"),
		})
		cursor = calendar.StartOfMonth(calendar.AddMonths(cursor, 1))
	}
	return ExpensePaceReport{Months: months, Currency: q.code()}
}

func (s Service) ExpensesByCategory(ctx context.Context, f Filter) ExpensesByCategoryReport {
	q := s.open(ctx, f)
	now := s.now()
	cur := q.byCategory("expense", f.Range(now))
	prev := lookup[int64, money.Money]{m: map[int64]money.Money{}, none: q.zero()}
	if cmp := f.Comparison(now); cmp != nil {
		for _, x := range q.byCategory("expense", *cmp) {
			prev.m[x.ID] = x.Total
		}
	}
	cats := []CategoryExpense{}
	for _, x := range cur {
		cats = append(cats, CategoryExpense{
			ID: x.ID, Name: x.Name, Icon: x.Icon, Color: x.Color,
			Current: x.Total, Previous: prev.at(x.ID),
		})
	}
	return ExpensesByCategoryReport{Categories: cats, Currency: q.code()}
}

// CategorySummary returns per-category totals of the given type for the period
// described by f, with the grand total.
func (s Service) CategorySummary(ctx context.Context, f Filter, typ string) ([]ledger.Category, money.Money) {
	q := s.open(ctx, f)
	out := []ledger.Category{}
	total := q.zero()
	for _, x := range q.byCategory(typ, f.Range(s.now())) {
		amount := x.Total
		out = append(out, ledger.Category{
			ID: x.ID, Name: x.Name, Type: typ,
			Icon: ptrTo(x.Icon), Color: ptrTo(x.Color), TotalAmount: &amount,
		})
		total = total.Add(amount)
	}
	return out, total
}

func ptrTo[T any](v T) *T { return &v }

type cashPoint struct {
	Date             string
	Income, Expenses money.Money
}

func (q report) cashFlow(r dateRange, groupBy string) []cashPoint {
	income, expenses := q.byPeriod("income", r, groupBy, 0), q.byPeriod("expense", r, groupBy, 0)
	periods := generatePeriods(r.Start, r.End, groupBy)
	out := make([]cashPoint, 0, len(periods))
	for _, p := range periods {
		out = append(out, cashPoint{Date: p.Key, Income: income.at(p.Key), Expenses: expenses.at(p.Key)})
	}
	return out
}

func (s Service) CashFlowOverTime(ctx context.Context, f Filter, groupBy string) CashFlowReport {
	q := s.open(ctx, f)
	now := s.now()
	current := q.cashFlow(f.Range(now), groupBy)
	var comparison []cashPoint
	if cmp := f.Comparison(now); cmp != nil {
		comparison = q.cashFlow(*cmp, groupBy)
	}
	items := []CashFlowPoint{}
	bal, prevBal := q.zero(), q.zero()
	for i, p := range current {
		bal = bal.Add(p.Income).Sub(p.Expenses)
		entry := CashFlowPoint{Date: p.Date, Income: p.Income, Expenses: p.Expenses, Balance: bal}
		if i < len(comparison) {
			c := comparison[i]
			prevBal = prevBal.Add(c.Income).Sub(c.Expenses)
			entry.PrevIncome, entry.PrevExpenses, entry.PrevBalance = &c.Income, &c.Expenses, ptrTo(prevBal)
		}
		items = append(items, entry)
	}
	return CashFlowReport{Items: items, Currency: q.code()}
}

func (s Service) Heatmap(ctx context.Context, f Filter) HeatmapReport {
	q := s.open(ctx, f)
	r := f.Range(s.now())
	daily := q.daily("expense", r)
	items := []HeatmapDay{}
	peak := q.zero()
	for d := r.Start; !d.After(r.End); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		row := daily.at(key)
		items = append(items, HeatmapDay{Date: key, Value: row.Total, Count: row.Count})
		if row.Total.Cmp(peak) > 0 {
			peak = row.Total
		}
	}
	return HeatmapReport{Items: items, Max: peak, Currency: q.code()}
}

func (s Service) TxSummary(ctx context.Context, f Filter, typ string) TxSummaryReport {
	q := s.open(ctx, f)
	now := s.now()
	r := f.Range(now)
	days := int(r.End.Sub(r.Start).Hours()/24) + 1
	total := q.sum(typ, r, 0)
	var previous, prevAvgDay, prevAvgWeek *money.Money
	if cmp := f.Comparison(now); cmp != nil {
		prev := q.sum(typ, *cmp, 0)
		previous = &prev
		if prevDays := int(cmp.End.Sub(cmp.Start).Hours()/24) + 1; prevDays > 0 {
			prevAvgDay, prevAvgWeek = ptrTo(q.average(prev, prevDays, 1)), ptrTo(q.average(prev, prevDays, 7))
		}
	}
	avgDay, avgWeek := q.zero(), q.zero()
	if days > 0 {
		avgDay, avgWeek = q.average(total, days, 1), q.average(total, days, 7)
	}
	return TxSummaryReport{
		Total: total, Previous: previous,
		AvgPerDay: avgDay, AvgPerWeek: avgWeek,
		PrevAvgPerDay: prevAvgDay, PrevAvgPerWeek: prevAvgWeek,
		DaysInPeriod: days, Currency: q.code(),
	}
}

func (s Service) TxByCategory(ctx context.Context, f Filter, typ string) TxByCategoryReport {
	q := s.open(ctx, f)
	cats := q.byCategory(typ, f.Range(s.now()))
	total := q.zero()
	for _, c := range cats {
		total = total.Add(c.Total)
	}
	items := []CategoryShare{}
	for _, c := range cats {
		pct := 0.0
		if total.IsPositive() {
			pct = percent(c.Total, total)
		}
		items = append(items, CategoryShare{
			ID: c.ID, Name: c.Name, Icon: c.Icon, Color: c.Color,
			Value: c.Total, Percentage: pct,
		})
	}
	return TxByCategoryReport{Items: items, Total: total, Currency: q.code()}
}

func (s Service) TxDynamics(ctx context.Context, f Filter, typ, groupBy string) TxDynamicsReport {
	q := s.open(ctx, f)
	r := f.Range(s.now())
	periods := generatePeriods(r.Start, r.End, groupBy)
	dates := make([]string, 0, len(periods))
	for _, p := range periods {
		dates = append(dates, p.Key)
	}
	series := func(categoryID int64) []money.Money {
		byPeriod := q.byPeriod(typ, r, groupBy, categoryID)
		data := make([]money.Money, 0, len(periods))
		for _, p := range periods {
			data = append(data, byPeriod.at(p.Key))
		}
		return data
	}
	color := "#22c55e"
	if typ == "expense" {
		color = "#ef4444"
	}
	datasets := []DynamicsSeries{{ID: 0, Name: "Total", Color: color, Data: series(0)}}
	for _, c := range q.byCategory(typ, r) {
		datasets = append(datasets, DynamicsSeries{ID: c.ID, Name: c.Name, Color: c.Color, Data: series(c.ID)})
	}
	return TxDynamicsReport{Dates: dates, Datasets: datasets, Currency: q.code()}
}

func (s Service) TxTop(ctx context.Context, f Filter, typ string, limit int) TxTopReport {
	if limit <= 0 {
		limit = 10
	}
	q := s.open(ctx, f)
	items := []TopTransaction{}
	for _, x := range filter.TopTransactions(ctx, s.DB, q.where(typ, f.Range(s.now()), 0), limit, q.unit) {
		var cat *TopCategory
		if x.CatID.Valid {
			cat = &TopCategory{
				ID: x.CatID.Int64, Name: x.CatName.String,
				Icon: coalesce(x.Icon.String, "circle"), Color: coalesce(x.Color.String, "#64748b"),
			}
		}
		items = append(items, TopTransaction{
			ID: x.ID, Description: nilOr(x.Description.String), Amount: x.Amount, Date: x.Date.String,
			Category: cat, Account: TopAccount{ID: x.AccID, Name: x.AccName.String},
		})
	}
	return TxTopReport{Items: items, Currency: q.code()}
}

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

// monthlyBudget is the monthly budget the report scope is measured against, in
// the base currency; nil when there is none. The budget limits of several
// currencies are converted at their rates and added before rounding.
func (q report) monthlyBudget() *money.Money {
	f := q.f
	scoped := len(f.CategoryIDs) > 0 || len(f.TagIDs) > 0
	if !scoped && len(f.AccountIDs) > 0 {
		return nil
	}
	var limits []filter.BudgetAmount
	switch {
	case scoped:
		limits = filter.ScopedMonthlyBudgets(q.ctx, q.DB, f.CategoryIDs, f.TagIDs)
	default:
		repo := ledgerrepo.Repo{DB: q.DB}
		if global := repo.GlobalMonthlyBudget(q.ctx); global != nil {
			// A global budget stands for the whole scope, whatever its size.
			return ptrTo(money.FromDecimal(budgetToBase(global.Amount, global.Rate, global.IsBase), q.unit))
		}
		rows, err := repo.MonthlyBudgets(q.ctx)
		if err != nil {
			return nil
		}
		for _, r := range rows {
			limits = append(limits, filter.BudgetAmount{Amount: r.Amount, Rate: r.Rate, IsBase: r.IsBase})
		}
	}
	total := decimal.Zero
	for _, b := range limits {
		total = total.Add(budgetToBase(b.Amount, b.Rate, b.IsBase))
	}
	if !total.IsPositive() {
		return nil
	}
	return ptrTo(money.FromDecimal(total, q.unit))
}

// budgetToBase is a budget limit in base-currency major units, unrounded.
func budgetToBase(limit money.Money, rate decimal.Decimal, isBase bool) decimal.Decimal {
	if isBase || rate.IsZero() {
		return limit.Decimal()
	}
	return limit.Decimal().Mul(rate)
}

func nilOr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
