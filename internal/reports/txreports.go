package reports

import (
	"context"

	"savvy-go/internal/db/filter"
	"savvy-go/internal/money"
)

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
			ID: x.ID, Description: ptrIfSet(x.Description.String), Amount: x.Amount, Date: x.Date.String,
			Category: cat, Account: TopAccount{ID: x.AccID, Name: x.AccName.String},
		})
	}
	return TxTopReport{Items: items, Currency: q.code()}
}
