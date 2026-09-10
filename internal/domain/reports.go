package domain

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/chiririll/savvy-plus/internal/db"
	"github.com/chiririll/savvy-plus/internal/db/filter"
)

type ReportFilter struct {
	PeriodType   string
	PeriodValue  string
	StartDate    string
	EndDate      string
	CompareWith  string
	AccountIDs   []int64
	CategoryIDs  []int64
	TagIDs       []int64
}

type dateRange struct {
	Start time.Time
	End   time.Time
}

type Reports struct {
	DB  *sql.DB
	Loc *time.Location
}

func (s Reports) loc() *time.Location {
	if s.Loc != nil {
		return s.Loc
	}
	return time.UTC
}

func (s Reports) now() time.Time { return time.Now().In(s.loc()) }

func (s Reports) baseCode(ctx context.Context) any {
	code, _ := db.Q(s.DB).GetBaseCurrencyCode(ctx)
	return nilOr(code)
}

func (s Reports) where(typ string, r dateRange, f ReportFilter, categoryID int64) filter.ReportWhere {
	return filter.ReportWhere{
		Type: typ, Start: r.Start.Format("2006-01-02"), End: r.End.Format("2006-01-02"),
		CategoryID: categoryID, AccountIDs: f.AccountIDs, CategoryIDs: f.CategoryIDs, TagIDs: f.TagIDs,
	}
}

func (f ReportFilter) Range(now time.Time) dateRange {
	switch f.PeriodType {
	case "month":
		t := now
		if y, m, ok := parseYearMonth(f.PeriodValue); ok {
			t = time.Date(y, time.Month(m), 1, 0, 0, 0, 0, now.Location())
		}
		return dateRange{startOfMonth(t), endOfMonth(t)}
	case "quarter":
		t := now
		if y, q, ok := parseYearQuarter(f.PeriodValue); ok {
			t = time.Date(y, time.Month((q-1)*3+1), 1, 0, 0, 0, 0, now.Location())
		}
		m := ((int(t.Month())-1)/3)*3 + 1
		start := time.Date(t.Year(), time.Month(m), 1, 0, 0, 0, 0, t.Location())
		return dateRange{start, endOfMonth(start.AddDate(0, 2, 0))}
	case "year":
		y := now.Year()
		if n, err := strconv.Atoi(f.PeriodValue); err == nil && n > 0 {
			y = n
		}
		return dateRange{time.Date(y, 1, 1, 0, 0, 0, 0, now.Location()), time.Date(y, 12, 31, 0, 0, 0, 0, now.Location())}
	case "ytd":
		return dateRange{time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()), dateOnly(now)}
	case "custom":
		start := startOfMonth(now)
		end := endOfMonth(now)
		if t, err := time.ParseInLocation("2006-01-02", f.StartDate, now.Location()); err == nil {
			start = t
		}
		if t, err := time.ParseInLocation("2006-01-02", f.EndDate, now.Location()); err == nil {
			end = t
		}
		return dateRange{start, end}
	default: // last_30_days
		end := dateOnly(now)
		return dateRange{end.AddDate(0, 0, -29), end}
	}
}

func (f ReportFilter) Comparison(now time.Time) *dateRange {
	if f.CompareWith == "" || f.CompareWith == "none" {
		return nil
	}
	cur := f.Range(now)
	if f.CompareWith == "same_period_last_year" {
		return &dateRange{cur.Start.AddDate(-1, 0, 0), cur.End.AddDate(-1, 0, 0)}
	}
	if f.CompareWith != "previous_period" {
		return nil
	}
	switch f.PeriodType {
	case "month":
		a := addMonthsNoOverflow(cur.Start, -1)
		return &dateRange{startOfMonth(a), endOfMonth(a)}
	case "quarter":
		a := addMonthsNoOverflow(cur.Start, -3)
		m := ((int(a.Month())-1)/3)*3 + 1
		start := time.Date(a.Year(), time.Month(m), 1, 0, 0, 0, 0, a.Location())
		return &dateRange{start, endOfMonth(start.AddDate(0, 2, 0))}
	case "year":
		return &dateRange{cur.Start.AddDate(-1, 0, 0), cur.End.AddDate(-1, 0, 0)}
	case "ytd":
		return &dateRange{cur.Start.AddDate(-1, 0, 0), cur.End.AddDate(-1, 0, 0)}
	default:
		days := int(cur.End.Sub(cur.Start).Hours()/24) + 1
		prevEnd := cur.Start.AddDate(0, 0, -1)
		return &dateRange{prevEnd.AddDate(0, 0, -(days - 1)), prevEnd}
	}
}

func (f ReportFilter) Sparkline(now time.Time, count int) []dateRange {
	cur := f.Range(now)
	days := int(cur.End.Sub(cur.Start).Hours()/24) + 1
	out := make([]dateRange, 0, count)
	for i := count - 1; i >= 0; i-- {
		switch f.PeriodType {
		case "quarter":
			a := addMonthsNoOverflow(cur.Start, -3*i)
			m := ((int(a.Month())-1)/3)*3 + 1
			start := time.Date(a.Year(), time.Month(m), 1, 0, 0, 0, 0, a.Location())
			out = append(out, dateRange{start, endOfMonth(start.AddDate(0, 2, 0))})
		case "year":
			a := cur.Start.AddDate(-i, 0, 0)
			out = append(out, dateRange{time.Date(a.Year(), 1, 1, 0, 0, 0, 0, a.Location()), time.Date(a.Year(), 12, 31, 0, 0, 0, 0, a.Location())})
		case "last_30_days", "custom":
			out = append(out, dateRange{cur.Start.AddDate(0, 0, -days*i), cur.End.AddDate(0, 0, -days*i)})
		case "ytd":
			a := addMonthsNoOverflow(cur.End, -i)
			out = append(out, dateRange{startOfMonth(a), endOfMonth(a)})
		default:
			a := addMonthsNoOverflow(cur.Start, -i)
			out = append(out, dateRange{startOfMonth(a), endOfMonth(a)})
		}
	}
	return out
}

func (s Reports) Overview(ctx context.Context, f ReportFilter) map[string]any {
	now := s.now()
	cur := s.metrics(ctx, f, f.Range(now))
	var prev *reportMetrics
	if cmp := f.Comparison(now); cmp != nil {
		m := s.metrics(ctx, f, *cmp)
		prev = &m
	}
	var incSpark, expSpark, netSpark, savSpark []float64
	for _, p := range f.Sparkline(now, 6) {
		m := s.metrics(ctx, f, p)
		incSpark = append(incSpark, m.Income)
		expSpark = append(expSpark, m.Expenses)
		netSpark = append(netSpark, m.Net)
		savSpark = append(savSpark, m.SavingsRate)
	}
	return map[string]any{
		"income":      metric(cur.Income, prevIncome(prev), incSpark),
		"expenses":    metric(cur.Expenses, prevExpenses(prev), expSpark),
		"netCashFlow": metric(cur.Net, prevNet(prev), netSpark),
		"savingsRate": metric(cur.SavingsRate, prevSavings(prev), savSpark),
		"currency":    s.baseCode(ctx),
	}
}

type reportMetrics struct {
	Income, Expenses, Net, SavingsRate float64
}

func (s Reports) metrics(ctx context.Context, f ReportFilter, r dateRange) reportMetrics {
	income := s.sumByType(ctx, "income", r, f, 0)
	expenses := s.sumByType(ctx, "expense", r, f, 0)
	net := income - expenses
	rate := 0.0
	switch {
	case income > 0:
		rate = math.Round((net/income)*1000) / 10
	case expenses > 0:
		rate = -100
	}
	return reportMetrics{round2(income), round2(expenses), round2(net), rate}
}

func metric(value float64, previous any, spark []float64) map[string]any {
	return map[string]any{"value": value, "previous": previous, "sparkline": spark}
}

func prevIncome(p *reportMetrics) any {
	if p == nil {
		return nil
	}
	return p.Income
}
func prevExpenses(p *reportMetrics) any {
	if p == nil {
		return nil
	}
	return p.Expenses
}
func prevNet(p *reportMetrics) any {
	if p == nil {
		return nil
	}
	return p.Net
}
func prevSavings(p *reportMetrics) any {
	if p == nil {
		return nil
	}
	return p.SavingsRate
}

func (s Reports) MoneyFlow(ctx context.Context, f ReportFilter) map[string]any {
	r := f.Range(s.now())
	income := s.sumGroupedByCategory(ctx, "income", r, f)
	expenses := s.sumGroupedByCategory(ctx, "expense", r, f)
	used := map[string]bool{}
	resolve := func(name string) string {
		c := name
		for used[c] {
			c += "\u200b"
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
	var totalInc, totalExp float64
	for _, x := range income {
		totalInc += x.Total
	}
	for _, x := range expenses {
		totalExp += x.Total
	}
	savings := totalInc - totalExp
	incomeColors := []string{"#22c55e", "#16a34a", "#15803d", "#14532d", "#166534", "#4ade80"}
	expenseColors := []string{"#ef4444", "#f97316", "#eab308", "#ec4899", "#8b5cf6", "#06b6d4", "#f43f5e", "#a855f7"}
	var nodes []map[string]any
	for i, x := range income {
		nodes = append(nodes, map[string]any{"name": x.Node, "itemStyle": map[string]any{"color": incomeColors[i%len(incomeColors)]}})
	}
	for i, x := range expenses {
		nodes = append(nodes, map[string]any{"name": x.Node, "itemStyle": map[string]any{"color": expenseColors[i%len(expenseColors)]}})
	}
	if savings > 0 {
		nodes = append(nodes, map[string]any{"name": savingsNode, "itemStyle": map[string]any{"color": "#3b82f6"}})
	}
	var links []map[string]any
	if totalInc > 0 && totalExp > 0 {
		for _, inc := range income {
			share := inc.Total / totalInc
			for _, exp := range expenses {
				if v := round2(share * exp.Total); v > 0 {
					links = append(links, map[string]any{"source": inc.Node, "target": exp.Node, "value": v})
				}
			}
			if savings > 0 {
				if v := round2(share * savings); v > 0 {
					links = append(links, map[string]any{"source": inc.Node, "target": savingsNode, "value": v})
				}
			}
		}
	}
	if nodes == nil {
		nodes = []map[string]any{}
	}
	if links == nil {
		links = []map[string]any{}
	}
	return map[string]any{
		"nodes": nodes, "links": links,
		"totals":   map[string]any{"income": round2(totalInc), "expenses": round2(totalExp), "savings": round2(math.Max(0, savings))},
		"currency": s.baseCode(ctx),
	}
}

func (s Reports) ExpensePace(ctx context.Context, f ReportFilter) map[string]any {
	now := s.now()
	r := f.Range(now)
	daily := s.dailyTotals(ctx, "expense", r, f)
	budget := s.monthlyBudget(ctx, f)
	today := dateOnly(now)
	var months []map[string]any
	cursor := startOfMonth(r.Start)
	for !cursor.After(r.End) {
		if cursor.After(today) && (cursor.Year() != today.Year() || cursor.Month() != today.Month()) {
			break
		}
		days := daysInMonth(cursor)
		var cumulative []float64
		total := 0.0
		for i := 0; i < days; i++ {
			day := cursor.AddDate(0, 0, i)
			if !day.Before(r.Start) && !day.After(r.End) {
				total += daily[day.Format("2006-01-02")].Total
			}
			cumulative = append(cumulative, round2(total))
		}
		var currentDay any
		if cursor.Year() == today.Year() && cursor.Month() == today.Month() {
			currentDay = today.Day()
		}
		spent := 0.0
		if len(cumulative) > 0 {
			spent = cumulative[len(cumulative)-1]
		}
		months = append(months, map[string]any{
			"label": cursor.Format("Jan 2006"), "budget": budget, "dailyExpenses": cumulative,
			"currentDay": currentDay, "daysInMonth": days, "totalSpent": spent,
			"monthStart": cursor.Format("2006-01-02"), "monthEnd": endOfMonth(cursor).Format("2006-01-02"),
		})
		cursor = startOfMonth(addMonthsNoOverflow(cursor, 1))
	}
	if months == nil {
		months = []map[string]any{}
	}
	return map[string]any{"months": months, "currency": s.baseCode(ctx)}
}

func (s Reports) ExpensesByCategory(ctx context.Context, f ReportFilter) map[string]any {
	now := s.now()
	cur := s.sumGroupedByCategory(ctx, "expense", f.Range(now), f)
	prevMap := map[int64]float64{}
	if cmp := f.Comparison(now); cmp != nil {
		for _, x := range s.sumGroupedByCategory(ctx, "expense", *cmp, f) {
			prevMap[x.ID] = x.Total
		}
	}
	var cats []map[string]any
	for _, x := range cur {
		cats = append(cats, map[string]any{
			"id": x.ID, "name": x.Name, "icon": x.Icon, "color": x.Color,
			"current": x.Total, "previous": prevMap[x.ID],
		})
	}
	if cats == nil {
		cats = []map[string]any{}
	}
	return map[string]any{"categories": cats, "currency": s.baseCode(ctx)}
}

func (s Reports) CashFlowOverTime(ctx context.Context, f ReportFilter, groupBy string) map[string]any {
	now := s.now()
	current := s.groupedCashFlow(ctx, f, f.Range(now), groupBy)
	var comparison []map[string]any
	if cmp := f.Comparison(now); cmp != nil {
		comparison = s.groupedCashFlow(ctx, f, *cmp, groupBy)
	}
	var items []map[string]any
	bal, prevBal := 0.0, 0.0
	for i, item := range current {
		inc, _ := item["income"].(float64)
		exp, _ := item["expenses"].(float64)
		bal += inc - exp
		entry := map[string]any{
			"label": item["label"], "date": item["date"],
			"income": inc, "expenses": exp, "balance": round2(bal),
		}
		if i < len(comparison) {
			pInc, _ := comparison[i]["income"].(float64)
			pExp, _ := comparison[i]["expenses"].(float64)
			prevBal += pInc - pExp
			entry["prevIncome"] = pInc
			entry["prevExpenses"] = pExp
			entry["prevBalance"] = round2(prevBal)
		}
		items = append(items, entry)
	}
	if items == nil {
		items = []map[string]any{}
	}
	return map[string]any{"items": items, "currency": s.baseCode(ctx)}
}

func (s Reports) groupedCashFlow(ctx context.Context, f ReportFilter, r dateRange, groupBy string) []map[string]any {
	income := s.groupedByPeriod(ctx, "income", r, f, groupBy, 0)
	expenses := s.groupedByPeriod(ctx, "expense", r, f, groupBy, 0)
	periods := generatePeriods(r.Start, r.End, groupBy)
	var out []map[string]any
	for _, p := range periods {
		out = append(out, map[string]any{
			"label": p.Label, "date": p.Key,
			"income": round2(income[p.Key]), "expenses": round2(expenses[p.Key]),
		})
	}
	return out
}

func (s Reports) Heatmap(ctx context.Context, f ReportFilter) map[string]any {
	r := f.Range(s.now())
	daily := s.dailyTotals(ctx, "expense", r, f)
	var items []map[string]any
	max := 0.0
	for d := r.Start; !d.After(r.End); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		row := daily[key]
		items = append(items, map[string]any{"date": key, "value": row.Total, "count": row.Count})
		if row.Total > max {
			max = row.Total
		}
	}
	if items == nil {
		items = []map[string]any{}
	}
	return map[string]any{"items": items, "max": round2(max), "currency": s.baseCode(ctx)}
}

func (s Reports) TxSummary(ctx context.Context, f ReportFilter, typ string) map[string]any {
	now := s.now()
	r := f.Range(now)
	days := int(r.End.Sub(r.Start).Hours()/24) + 1
	total := s.sumByType(ctx, typ, r, f, 0)
	var previous any
	var prevAvgDay, prevAvgWeek any
	if cmp := f.Comparison(now); cmp != nil {
		prev := s.sumByType(ctx, typ, *cmp, f, 0)
		previous = round2(prev)
		prevDays := int(cmp.End.Sub(cmp.Start).Hours()/24) + 1
		if prevDays > 0 {
			prevAvgDay = round2(prev / float64(prevDays))
			prevAvgWeek = round2(prev / (float64(prevDays) / 7))
		}
	}
	avgDay, avgWeek := 0.0, 0.0
	if days > 0 {
		avgDay = round2(total / float64(days))
		avgWeek = round2(total / (float64(days) / 7))
	}
	return map[string]any{
		"total": round2(total), "previous": previous,
		"avgPerDay": avgDay, "avgPerWeek": avgWeek,
		"prevAvgPerDay": prevAvgDay, "prevAvgPerWeek": prevAvgWeek,
		"daysInPeriod": days, "currency": s.baseCode(ctx),
	}
}

func (s Reports) TxByCategory(ctx context.Context, f ReportFilter, typ string) map[string]any {
	cats := s.sumGroupedByCategory(ctx, typ, f.Range(s.now()), f)
	var total float64
	for _, c := range cats {
		total += c.Total
	}
	var items []map[string]any
	for _, c := range cats {
		pct := 0.0
		if total > 0 {
			pct = math.Round((c.Total/total)*1000) / 10
		}
		items = append(items, map[string]any{
			"id": c.ID, "name": c.Name, "icon": c.Icon, "color": c.Color,
			"value": c.Total, "percentage": pct,
		})
	}
	if items == nil {
		items = []map[string]any{}
	}
	return map[string]any{"items": items, "total": round2(total), "currency": s.baseCode(ctx)}
}

func (s Reports) TxDynamics(ctx context.Context, f ReportFilter, typ, groupBy string) map[string]any {
	r := f.Range(s.now())
	cats := s.sumGroupedByCategory(ctx, typ, r, f)
	periods := generatePeriods(r.Start, r.End, groupBy)
	var labels, dates []string
	for _, p := range periods {
		labels = append(labels, p.Label)
		dates = append(dates, p.Key)
	}
	totals := s.groupedByPeriod(ctx, typ, r, f, groupBy, 0)
	var totalData []float64
	for _, p := range periods {
		totalData = append(totalData, round2(totals[p.Key]))
	}
	color := "#22c55e"
	if typ == "expense" {
		color = "#ef4444"
	}
	datasets := []map[string]any{{"id": 0, "name": "Total", "color": color, "data": totalData}}
	for _, c := range cats {
		byPeriod := s.groupedByPeriod(ctx, typ, r, f, groupBy, c.ID)
		var data []float64
		for _, p := range periods {
			data = append(data, round2(byPeriod[p.Key]))
		}
		datasets = append(datasets, map[string]any{"id": c.ID, "name": c.Name, "color": c.Color, "data": data})
	}
	if labels == nil {
		labels = []string{}
	}
	if dates == nil {
		dates = []string{}
	}
	return map[string]any{"labels": labels, "dates": dates, "datasets": datasets, "currency": s.baseCode(ctx)}
}

func (s Reports) TxTop(ctx context.Context, f ReportFilter, typ string, limit int) map[string]any {
	if limit <= 0 {
		limit = 10
	}
	r := f.Range(s.now())
	rows := filter.TopTransactions(ctx, s.DB, s.where(typ, r, f, 0), limit)
	var items []map[string]any
	for _, x := range rows {
		var cat any
		if x.CatID.Valid {
			cat = map[string]any{
				"id": x.CatID.Int64, "name": x.CatName.String,
				"icon": coalesce(x.Icon.String, "circle"), "color": coalesce(x.Color.String, "#64748b"),
			}
		}
		items = append(items, map[string]any{
			"id": x.ID, "description": nilOr(x.Description.String), "amount": round2(x.Amount), "date": x.Date.String,
			"category": cat, "account": map[string]any{"id": x.AccID, "name": x.AccName.String},
		})
	}
	if items == nil {
		items = []map[string]any{}
	}
	return map[string]any{"items": items, "currency": s.baseCode(ctx)}
}

func (s Reports) NetWorth(ctx context.Context, f ReportFilter) map[string]any {
	now := s.now()
	current := s.netWorthAt(ctx, f.Range(now).End, f)
	var curTotal float64
	for _, a := range current {
		curTotal += a.Balance
	}
	var previous any
	change := 0.0
	if cmp := f.Comparison(now); cmp != nil {
		prev := s.netWorthAt(ctx, cmp.End, f)
		var prevTotal float64
		for _, a := range prev {
			prevTotal += a.Balance
		}
		previous = round2(prevTotal)
		change = curTotal - prevTotal
	}
	changePct := 0.0
	if prev, ok := previous.(float64); ok && prev != 0 {
		changePct = math.Round((change/math.Abs(prev))*1000) / 10
	}
	var accounts []map[string]any
	for _, a := range current {
		pct := 0.0
		if curTotal > 0 {
			pct = math.Round((a.Balance/curTotal)*1000) / 10
		}
		accounts = append(accounts, map[string]any{
			"id": a.ID, "name": a.Name, "type": a.Type,
			"balance": round2(a.Balance), "percentage": pct,
		})
	}
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i]["balance"].(float64) > accounts[j]["balance"].(float64)
	})
	if accounts == nil {
		accounts = []map[string]any{}
	}
	return map[string]any{
		"current": round2(curTotal), "previous": previous,
		"change": round2(change), "changePercent": changePct,
		"accounts": accounts, "currency": s.baseCode(ctx),
	}
}

func (s Reports) NetWorthHistory(ctx context.Context, f ReportFilter, groupBy string) map[string]any {
	r := f.Range(s.now())
	var labels, dates []string
	var values []float64
	for cur := r.Start; !cur.After(r.End); cur = nextPeriod(cur, groupBy) {
		end := periodEnd(cur, r.End, groupBy)
		labels = append(labels, reportPeriodLabel(cur, groupBy))
		dates = append(dates, cur.Format("2006-01-02"))
		var total float64
		for _, a := range s.netWorthAt(ctx, end, f) {
			total += a.Balance
		}
		values = append(values, round2(total))
	}
	if labels == nil {
		labels = []string{}
		dates = []string{}
		values = []float64{}
	}
	return map[string]any{"labels": labels, "dates": dates, "values": values, "currency": s.baseCode(ctx)}
}

type nwAccount struct {
	ID      int64
	Name    string
	Type    string
	Balance float64
}

func (s Reports) netWorthAt(ctx context.Context, at time.Time, f ReportFilter) []nwAccount {
	accts := Accounts{DB: s.DB}
	list, err := accts.All(ctx, true, true)
	if err != nil {
		return nil
	}
	want := map[int64]bool{}
	for _, id := range f.AccountIDs {
		want[id] = true
	}
	cutoff := at.Format("2006-01-02")
	var out []nwAccount
	for _, a := range list {
		if len(want) > 0 && !want[a.ID] {
			continue
		}
		bal, _ := accts.balance(ctx, a, cutoff)
		rate := 1.0
		if a.Currency != nil {
			rate = a.Currency.Rate
			if rate == 0 {
				rate = 1
			}
		}
		out = append(out, nwAccount{ID: a.ID, Name: a.Name, Type: a.Type, Balance: bal * rate})
	}
	return out
}

type catTotal struct {
	ID    int64
	Name  string
	Icon  string
	Color string
	Total float64
	Node  string
}

type dayTotal struct {
	Total float64
	Count int
}

func (s Reports) sumByType(ctx context.Context, typ string, r dateRange, f ReportFilter, categoryID int64) float64 {
	return filter.SumByType(ctx, s.DB, s.where(typ, r, f, categoryID))
}

func (s Reports) sumGroupedByCategory(ctx context.Context, typ string, r dateRange, f ReportFilter) []catTotal {
	rows := filter.SumGroupedByCategory(ctx, s.DB, s.where(typ, r, f, 0))
	var out []catTotal
	for _, x := range rows {
		out = append(out, catTotal{
			ID: x.ID, Name: x.Name, Icon: coalesce(x.Icon.String, "circle"),
			Color: coalesce(x.Color.String, "#64748b"), Total: round2(x.Total),
		})
	}
	return out
}

func (s Reports) dailyTotals(ctx context.Context, typ string, r dateRange, f ReportFilter) map[string]dayTotal {
	out := map[string]dayTotal{}
	for _, x := range filter.DailyTotals(ctx, s.DB, s.where(typ, r, f, 0)) {
		out[x.Day] = dayTotal{Total: round2(x.Total), Count: x.Count}
	}
	return out
}

func (s Reports) groupedByPeriod(ctx context.Context, typ string, r dateRange, f ReportFilter, groupBy string, categoryID int64) map[string]float64 {
	out := map[string]float64{}
	for _, x := range filter.GroupedByPeriod(ctx, s.DB, s.where(typ, r, f, categoryID), groupBy) {
		out[x.Key] = x.Total
	}
	return out
}

func (s Reports) monthlyBudget(ctx context.Context, f ReportFilter) any {
	scoped := len(f.CategoryIDs) > 0 || len(f.TagIDs) > 0
	if !scoped && len(f.AccountIDs) > 0 {
		return nil
	}
	if !scoped {
		row, err := db.Q(s.DB).GetGlobalMonthlyBudget(ctx)
		if err == nil {
			return budgetToBase(row.Amount, nullRate(row.Rate), row.IsBase.Valid && row.IsBase.Int64 != 0)
		}
		rows, err := db.Q(s.DB).ListMonthlyBudgets(ctx)
		if err != nil {
			return nil
		}
		total := 0.0
		for _, r := range rows {
			total += budgetToBase(r.Amount, nullRate(r.Rate), r.IsBase.Valid && r.IsBase.Int64 != 0)
		}
		if total > 0 {
			return total
		}
		return nil
	}
	rows := filter.ScopedMonthlyBudgets(ctx, s.DB, f.CategoryIDs, f.TagIDs)
	total := 0.0
	for _, r := range rows {
		total += budgetToBase(r.Amount, r.Rate, r.IsBase)
	}
	if total > 0 {
		return total
	}
	return nil
}

func nullRate(v sql.NullFloat64) float64 {
	return v.Float64
}

func budgetToBase(amount, rate float64, isBase bool) float64 {
	if isBase || rate == 0 {
		return amount
	}
	return amount * rate
}


type periodPoint struct{ Key, Label string }

func generatePeriods(start, end time.Time, groupBy string) []periodPoint {
	var out []periodPoint
	cur := start
	for !cur.After(end) {
		switch groupBy {
		case "week":
			wk := startOfWeekSunday(cur)
			out = append(out, periodPoint{Key: wk.Format("2006-01-02"), Label: "Week " + wk.Format("Jan 2")})
			cur = cur.AddDate(0, 0, 7)
		case "month":
			m := startOfMonth(cur)
			out = append(out, periodPoint{Key: m.Format("2006-01-02"), Label: m.Format("Jan '06")})
			cur = addMonthsNoOverflow(cur, 1)
		default:
			out = append(out, periodPoint{Key: cur.Format("2006-01-02"), Label: cur.Format("Jan 2")})
			cur = cur.AddDate(0, 0, 1)
		}
	}
	return out
}

func nextPeriod(cur time.Time, groupBy string) time.Time {
	switch groupBy {
	case "week":
		return startOfWeek(cur.AddDate(0, 0, 7))
	case "month":
		return startOfMonth(addMonthsNoOverflow(cur, 1))
	default:
		return cur.AddDate(0, 0, 1)
	}
}

func periodEnd(cur, max time.Time, groupBy string) time.Time {
	var end time.Time
	switch groupBy {
	case "week":
		end = endOfWeek(cur)
	case "month":
		end = endOfMonth(cur)
	default:
		end = cur
	}
	if end.After(max) {
		return max
	}
	return end
}

func reportPeriodLabel(cur time.Time, groupBy string) string {
	switch groupBy {
	case "week":
		_, week := cur.ISOWeek()
		return fmt.Sprintf("W%d %s", week, cur.Format("Jan '06"))
	case "month":
		return cur.Format("Jan '06")
	default:
		return cur.Format("Jan 2")
	}
}

func startOfWeekSunday(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day()-int(t.Weekday()), 0, 0, 0, 0, t.Location())
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func parseYearMonth(s string) (int, int, bool) {
	if len(s) != 7 || s[4] != '-' {
		return 0, 0, false
	}
	y, err1 := strconv.Atoi(s[:4])
	m, err2 := strconv.Atoi(s[5:])
	return y, m, err1 == nil && err2 == nil && m >= 1 && m <= 12
}

func parseYearQuarter(s string) (int, int, bool) {
	if len(s) < 7 || s[4] != '-' || s[5] != 'Q' {
		return 0, 0, false
	}
	y, err1 := strconv.Atoi(s[:4])
	q, err2 := strconv.Atoi(s[6:])
	return y, q, err1 == nil && err2 == nil && q >= 1 && q <= 4
}

func coalesce(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
