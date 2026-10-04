package domain

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db"
	"savvy-go/internal/db/filter"
	"savvy-go/internal/money"
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
	Reports
	ctx  context.Context
	f    ReportFilter
	base *Currency  // nil when there is no currency yet
	unit money.Unit // the unit of base; the zero Unit without one
}

func (s Reports) open(ctx context.Context, f ReportFilter) report {
	q := report{Reports: s, ctx: ctx, f: f}
	if base, _ := (Currencies{DB: s.DB}).Base(ctx); base != nil {
		q.base, q.unit = base, base.Unit()
	}
	return q
}

func (q report) zero() money.Money { return money.Zero(q.unit) }

// code is the base currency code for the response; null without a base.
func (q report) code() any {
	if q.base == nil {
		return nil
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

func (s Reports) Overview(ctx context.Context, f ReportFilter) map[string]any {
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
	return map[string]any{
		"income":      metric(cur.Income, prevMetric(prev, func(m reportMetrics) money.Money { return m.Income }), incSpark),
		"expenses":    metric(cur.Expenses, prevMetric(prev, func(m reportMetrics) money.Money { return m.Expenses }), expSpark),
		"netCashFlow": metric(cur.Net, prevMetric(prev, func(m reportMetrics) money.Money { return m.Net }), netSpark),
		"savingsRate": metric(cur.SavingsRate, prevMetric(prev, func(m reportMetrics) float64 { return m.SavingsRate }), savSpark),
		"currency":    q.code(),
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

func metric(value, previous, sparkline any) map[string]any {
	return map[string]any{"value": value, "previous": previous, "sparkline": sparkline}
}

// prevMetric is a metric of the comparison period; null without one.
func prevMetric[T any](p *reportMetrics, get func(reportMetrics) T) any {
	if p == nil {
		return nil
	}
	return get(*p)
}

func (s Reports) MoneyFlow(ctx context.Context, f ReportFilter) map[string]any {
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
	nodes := []map[string]any{}
	for i, x := range income {
		nodes = append(nodes, map[string]any{"name": x.Node, "itemStyle": map[string]any{"color": incomeColors[i%len(incomeColors)]}})
	}
	for i, x := range expenses {
		nodes = append(nodes, map[string]any{"name": x.Node, "itemStyle": map[string]any{"color": expenseColors[i%len(expenseColors)]}})
	}
	if savings.IsPositive() {
		nodes = append(nodes, map[string]any{"name": savingsNode, "itemStyle": map[string]any{"color": "#3b82f6"}})
	}
	links := []map[string]any{}
	if totalInc.IsPositive() && totalExp.IsPositive() {
		for _, inc := range income {
			for _, exp := range expenses {
				if v := q.portion(exp.Total, inc.Total, totalInc); v.IsPositive() {
					links = append(links, map[string]any{"source": inc.Node, "target": exp.Node, "value": v})
				}
			}
			if savings.IsPositive() {
				if v := q.portion(savings, inc.Total, totalInc); v.IsPositive() {
					links = append(links, map[string]any{"source": inc.Node, "target": savingsNode, "value": v})
				}
			}
		}
	}
	return map[string]any{
		"nodes": nodes, "links": links,
		"totals":   map[string]any{"income": totalInc, "expenses": totalExp, "savings": money.Max(q.zero(), savings)},
		"currency": q.code(),
	}
}

func (s Reports) ExpensePace(ctx context.Context, f ReportFilter) map[string]any {
	q := s.open(ctx, f)
	now := s.now()
	r := f.Range(now)
	daily := q.daily("expense", r)
	budget := q.monthlyBudget()
	today := dateOnly(now)
	months := []map[string]any{}
	cursor := startOfMonth(r.Start)
	for !cursor.After(r.End) {
		if cursor.After(today) && (cursor.Year() != today.Year() || cursor.Month() != today.Month()) {
			break
		}
		days := daysInMonth(cursor)
		cumulative := make([]money.Money, 0, days)
		total := q.zero()
		for i := 0; i < days; i++ {
			day := cursor.AddDate(0, 0, i)
			if !day.Before(r.Start) && !day.After(r.End) {
				total = total.Add(daily.at(day.Format("2006-01-02")).Total)
			}
			cumulative = append(cumulative, total)
		}
		var currentDay any
		if cursor.Year() == today.Year() && cursor.Month() == today.Month() {
			currentDay = today.Day()
		}
		months = append(months, map[string]any{
			"budget": budget, "dailyExpenses": cumulative,
			"currentDay": currentDay, "daysInMonth": days, "totalSpent": total,
			"monthStart": cursor.Format("2006-01-02"), "monthEnd": endOfMonth(cursor).Format("2006-01-02"),
		})
		cursor = startOfMonth(addMonthsNoOverflow(cursor, 1))
	}
	return map[string]any{"months": months, "currency": q.code()}
}

func (s Reports) ExpensesByCategory(ctx context.Context, f ReportFilter) map[string]any {
	q := s.open(ctx, f)
	now := s.now()
	cur := q.byCategory("expense", f.Range(now))
	prev := lookup[int64, money.Money]{m: map[int64]money.Money{}, none: q.zero()}
	if cmp := f.Comparison(now); cmp != nil {
		for _, x := range q.byCategory("expense", *cmp) {
			prev.m[x.ID] = x.Total
		}
	}
	cats := []map[string]any{}
	for _, x := range cur {
		cats = append(cats, map[string]any{
			"id": x.ID, "name": x.Name, "icon": x.Icon, "color": x.Color,
			"current": x.Total, "previous": prev.at(x.ID),
		})
	}
	return map[string]any{"categories": cats, "currency": q.code()}
}

// CategorySummary returns per-category totals of the given type for the period
// described by f, with the grand total.
func (s Reports) CategorySummary(ctx context.Context, f ReportFilter, typ string) ([]Category, money.Money) {
	q := s.open(ctx, f)
	out := []Category{}
	total := q.zero()
	for _, x := range q.byCategory(typ, f.Range(s.now())) {
		amount := x.Total
		out = append(out, Category{
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

func (s Reports) CashFlowOverTime(ctx context.Context, f ReportFilter, groupBy string) map[string]any {
	q := s.open(ctx, f)
	now := s.now()
	current := q.cashFlow(f.Range(now), groupBy)
	var comparison []cashPoint
	if cmp := f.Comparison(now); cmp != nil {
		comparison = q.cashFlow(*cmp, groupBy)
	}
	items := []map[string]any{}
	bal, prevBal := q.zero(), q.zero()
	for i, p := range current {
		bal = bal.Add(p.Income).Sub(p.Expenses)
		entry := map[string]any{"date": p.Date, "income": p.Income, "expenses": p.Expenses, "balance": bal}
		if i < len(comparison) {
			c := comparison[i]
			prevBal = prevBal.Add(c.Income).Sub(c.Expenses)
			entry["prevIncome"], entry["prevExpenses"], entry["prevBalance"] = c.Income, c.Expenses, prevBal
		}
		items = append(items, entry)
	}
	return map[string]any{"items": items, "currency": q.code()}
}

func (s Reports) Heatmap(ctx context.Context, f ReportFilter) map[string]any {
	q := s.open(ctx, f)
	r := f.Range(s.now())
	daily := q.daily("expense", r)
	items := []map[string]any{}
	peak := q.zero()
	for d := r.Start; !d.After(r.End); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		row := daily.at(key)
		items = append(items, map[string]any{"date": key, "value": row.Total, "count": row.Count})
		if row.Total.Cmp(peak) > 0 {
			peak = row.Total
		}
	}
	return map[string]any{"items": items, "max": peak, "currency": q.code()}
}

func (s Reports) TxSummary(ctx context.Context, f ReportFilter, typ string) map[string]any {
	q := s.open(ctx, f)
	now := s.now()
	r := f.Range(now)
	days := int(r.End.Sub(r.Start).Hours()/24) + 1
	total := q.sum(typ, r, 0)
	var previous, prevAvgDay, prevAvgWeek any
	if cmp := f.Comparison(now); cmp != nil {
		prev := q.sum(typ, *cmp, 0)
		previous = prev
		if prevDays := int(cmp.End.Sub(cmp.Start).Hours()/24) + 1; prevDays > 0 {
			prevAvgDay, prevAvgWeek = q.average(prev, prevDays, 1), q.average(prev, prevDays, 7)
		}
	}
	avgDay, avgWeek := q.zero(), q.zero()
	if days > 0 {
		avgDay, avgWeek = q.average(total, days, 1), q.average(total, days, 7)
	}
	return map[string]any{
		"total": total, "previous": previous,
		"avgPerDay": avgDay, "avgPerWeek": avgWeek,
		"prevAvgPerDay": prevAvgDay, "prevAvgPerWeek": prevAvgWeek,
		"daysInPeriod": days, "currency": q.code(),
	}
}

func (s Reports) TxByCategory(ctx context.Context, f ReportFilter, typ string) map[string]any {
	q := s.open(ctx, f)
	cats := q.byCategory(typ, f.Range(s.now()))
	total := q.zero()
	for _, c := range cats {
		total = total.Add(c.Total)
	}
	items := []map[string]any{}
	for _, c := range cats {
		pct := 0.0
		if total.IsPositive() {
			pct = percent(c.Total, total)
		}
		items = append(items, map[string]any{
			"id": c.ID, "name": c.Name, "icon": c.Icon, "color": c.Color,
			"value": c.Total, "percentage": pct,
		})
	}
	return map[string]any{"items": items, "total": total, "currency": q.code()}
}

func (s Reports) TxDynamics(ctx context.Context, f ReportFilter, typ, groupBy string) map[string]any {
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
	datasets := []map[string]any{{"id": 0, "name": "Total", "color": color, "data": series(0)}}
	for _, c := range q.byCategory(typ, r) {
		datasets = append(datasets, map[string]any{"id": c.ID, "name": c.Name, "color": c.Color, "data": series(c.ID)})
	}
	return map[string]any{"dates": dates, "datasets": datasets, "currency": q.code()}
}

func (s Reports) TxTop(ctx context.Context, f ReportFilter, typ string, limit int) map[string]any {
	if limit <= 0 {
		limit = 10
	}
	q := s.open(ctx, f)
	items := []map[string]any{}
	for _, x := range filter.TopTransactions(ctx, s.DB, q.where(typ, f.Range(s.now()), 0), limit, q.unit) {
		var cat any
		if x.CatID.Valid {
			cat = map[string]any{
				"id": x.CatID.Int64, "name": x.CatName.String,
				"icon": coalesce(x.Icon.String, "circle"), "color": coalesce(x.Color.String, "#64748b"),
			}
		}
		items = append(items, map[string]any{
			"id": x.ID, "description": nilOr(x.Description.String), "amount": x.Amount, "date": x.Date.String,
			"category": cat, "account": map[string]any{"id": x.AccID, "name": x.AccName.String},
		})
	}
	return map[string]any{"items": items, "currency": q.code()}
}

type nwAccount struct {
	ID      int64
	Name    string
	Type    string
	Balance money.Money // in the base currency
}

// accounts are the active, non-debt accounts the report covers.
func (q report) accounts() []Account {
	list, err := Accounts{DB: q.DB}.All(q.ctx, true, true)
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
func (q report) netWorthAt(accts []Account, at time.Time) []nwAccount {
	cutoff := at.Format("2006-01-02")
	out := make([]nwAccount, 0, len(accts))
	for _, a := range accts {
		bal, _ := Accounts{DB: q.DB}.balance(q.ctx, a, cutoff)
		out = append(out, nwAccount{ID: a.ID, Name: a.Name, Type: a.Type, Balance: Convert(bal, *a.Currency, *q.base)})
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

func (s Reports) NetWorth(ctx context.Context, f ReportFilter) map[string]any {
	q := s.open(ctx, f)
	now := s.now()
	accts := q.accounts()
	current := q.netWorthAt(accts, f.Range(now).End)
	curTotal := q.totalOf(current)
	var previous any
	change := q.zero()
	changePct := 0.0
	if cmp := f.Comparison(now); cmp != nil {
		prevTotal := q.totalOf(q.netWorthAt(accts, cmp.End))
		previous = prevTotal
		change = curTotal.Sub(prevTotal)
		if !prevTotal.IsZero() {
			changePct = percent(change, prevTotal.Abs())
		}
	}
	sort.SliceStable(current, func(i, j int) bool { return current[i].Balance.Cmp(current[j].Balance) > 0 })
	rows := []map[string]any{}
	for _, a := range current {
		pct := 0.0
		if curTotal.IsPositive() {
			pct = percent(a.Balance, curTotal)
		}
		rows = append(rows, map[string]any{"id": a.ID, "name": a.Name, "type": a.Type, "balance": a.Balance, "percentage": pct})
	}
	return map[string]any{
		"current": curTotal, "previous": previous,
		"change": change, "changePercent": changePct,
		"accounts": rows, "currency": q.code(),
	}
}

func (s Reports) NetWorthHistory(ctx context.Context, f ReportFilter, groupBy string) map[string]any {
	q := s.open(ctx, f)
	r := f.Range(s.now())
	accts := q.accounts()
	dates, values := []string{}, []money.Money{}
	for cur := r.Start; !cur.After(r.End); cur = nextPeriod(cur, groupBy) {
		dates = append(dates, cur.Format("2006-01-02"))
		values = append(values, q.totalOf(q.netWorthAt(accts, periodEnd(cur, r.End, groupBy))))
	}
	return map[string]any{"dates": dates, "values": values, "currency": q.code()}
}

// monthlyBudget is the monthly budget the report scope is measured against, in
// the base currency; null when there is none. The budget limits of several
// currencies are converted at their rates and added before rounding.
func (q report) monthlyBudget() any {
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
		if row, err := db.Q(q.DB).GetGlobalMonthlyBudget(q.ctx); err == nil {
			// A global budget stands for the whole scope, whatever its size.
			amount := money.New(row.Amount, money.Unit{ID: row.CurrencyID, Decimals: int(row.Decimals)})
			return money.FromDecimal(budgetToBase(amount, row.Rate, row.IsBase != 0), q.unit)
		}
		rows, err := db.Q(q.DB).ListMonthlyBudgets(q.ctx)
		if err != nil {
			return nil
		}
		for _, r := range rows {
			amount := money.New(r.Amount, money.Unit{ID: r.CurrencyID, Decimals: int(r.Decimals)})
			limits = append(limits, filter.BudgetAmount{Amount: amount, Rate: r.Rate, IsBase: r.IsBase != 0})
		}
	}
	total := decimal.Zero
	for _, b := range limits {
		total = total.Add(budgetToBase(b.Amount, b.Rate, b.IsBase))
	}
	if !total.IsPositive() {
		return nil
	}
	return money.FromDecimal(total, q.unit)
}

// budgetToBase is a budget limit in base-currency major units, unrounded.
func budgetToBase(limit money.Money, rate decimal.Decimal, isBase bool) decimal.Decimal {
	if isBase || rate.IsZero() {
		return limit.Decimal()
	}
	return limit.Decimal().Mul(rate)
}
