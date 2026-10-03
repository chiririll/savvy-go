package domain

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db"
	"savvy-go/internal/db/filter"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/money"
)

type BudgetProgress struct {
	Spent       decimal.Decimal
	Remaining   decimal.Decimal
	Percent     float64
	PeriodStart string
	PeriodEnd   string
	IsExceeded  bool
}

type Budget struct {
	ID              int64
	Name            string
	Amount          decimal.Decimal
	CurrencyID      *int64
	Period          string
	StartDate       *string
	EndDate         *string
	IsGlobal        bool
	NotifyAtPercent *int
	IsActive        bool
	Currency        *Currency
	Categories      []Category
	Tags            []Tag
	Progress        *BudgetProgress
}

type BudgetInput struct {
	Name            string
	Amount          decimal.Decimal
	CurrencyID      *int64
	Period          string
	StartDate       *string
	EndDate         *string
	IsGlobal        *bool
	NotifyAtPercent *int
	IsActive        *bool
	CategoryIDs     []int64
	TagIDs          []int64
	HasCategoryIDs  bool
	HasTagIDs       bool
}

type Budgets struct{ DB *sql.DB }

func (s Budgets) All(ctx context.Context) ([]Budget, error) {
	return s.list(ctx, sql.NullInt64{})
}

func (s Budgets) ByID(ctx context.Context, id int64) (*Budget, error) {
	list, err := s.list(ctx, db.NI(id))
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func (s Budgets) Create(ctx context.Context, in BudgetInput) (*Budget, error) {
	active := true
	if in.IsActive != nil {
		active = *in.IsActive
	}
	global := false
	if in.IsGlobal != nil {
		global = *in.IsGlobal
	}
	now := time.Now().UTC().Format(time.RFC3339)
	currencyID, dec, err := s.resolveCurrency(ctx, in.CurrencyID)
	if err != nil {
		return nil, err
	}
	res, err := db.Q(s.DB).InsertBudget(ctx, sqlc.InsertBudgetParams{
		Name: in.Name, Amount: money.ToMinor(in.Amount, dec), CurrencyID: currencyID, Period: in.Period,
		StartDate: db.NullString(in.StartDate), EndDate: db.NullString(in.EndDate),
		IsGlobal: db.BoolInt(global), NotifyAtPercent: db.NullInt(in.NotifyAtPercent),
		IsActive: db.BoolInt(active), CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
	})
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	if err := s.saveCats(ctx, id, in.CategoryIDs); err != nil {
		return nil, err
	}
	if err := s.saveTags(ctx, id, in.TagIDs); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Budgets) Update(ctx context.Context, id int64, in BudgetInput) (*Budget, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if in.Name == "" {
		in.Name = cur.Name
	}
	if in.Amount.IsZero() {
		in.Amount = cur.Amount
	}
	if in.Period == "" {
		in.Period = cur.Period
	}
	if in.CurrencyID == nil {
		in.CurrencyID = cur.CurrencyID
	}
	if in.StartDate == nil {
		in.StartDate = cur.StartDate
	}
	if in.EndDate == nil {
		in.EndDate = cur.EndDate
	}
	global := cur.IsGlobal
	if in.IsGlobal != nil {
		global = *in.IsGlobal
	}
	active := cur.IsActive
	if in.IsActive != nil {
		active = *in.IsActive
	}
	notify := cur.NotifyAtPercent
	if in.NotifyAtPercent != nil {
		notify = in.NotifyAtPercent
	}
	now := time.Now().UTC().Format(time.RFC3339)
	currencyID, dec, err := s.resolveCurrency(ctx, in.CurrencyID)
	if err != nil {
		return nil, err
	}
	err = db.Q(s.DB).UpdateBudget(ctx, sqlc.UpdateBudgetParams{
		Name: in.Name, Amount: money.ToMinor(in.Amount, dec), CurrencyID: currencyID, Period: in.Period,
		StartDate: db.NullString(in.StartDate), EndDate: db.NullString(in.EndDate),
		IsGlobal: db.BoolInt(global), NotifyAtPercent: db.NullInt(notify),
		IsActive: db.BoolInt(active), UpdatedAt: db.NS(now), ID: id,
	})
	if err != nil {
		return nil, err
	}
	if in.HasCategoryIDs {
		if err := s.saveCats(ctx, id, in.CategoryIDs); err != nil {
			return nil, err
		}
	}
	if in.HasTagIDs {
		if err := s.saveTags(ctx, id, in.TagIDs); err != nil {
			return nil, err
		}
	}
	return s.ByID(ctx, id)
}

func (s Budgets) Delete(ctx context.Context, id int64) error {
	return db.Q(s.DB).DeleteBudget(ctx, id)
}

func (s Budgets) CalculateProgress(ctx context.Context, b *Budget) BudgetProgress {
	start, end := budgetPeriodDates(*b)
	spent := s.spent(ctx, b, start, end)
	remaining := decimal.Max(decimal.Zero, b.Amount.Sub(spent))
	percent := 0.0
	if b.Amount.IsPositive() {
		percent = math.Round(spent.Div(b.Amount).InexactFloat64()*1000) / 10
	}
	return BudgetProgress{
		Spent: spent, Remaining: remaining, Percent: percent,
		PeriodStart: start.Format("2006-01-02"), PeriodEnd: end.Format("2006-01-02"),
		IsExceeded: spent.GreaterThan(b.Amount),
	}
}

func (s Budgets) spent(ctx context.Context, b *Budget, start, end time.Time) decimal.Decimal {
	targetRate := one
	if b.Currency != nil && b.Currency.Rate.IsPositive() {
		targetRate = b.Currency.Rate
	}

	var catIDs []int64
	if !b.IsGlobal {
		if len(b.Categories) == 0 {
			return decimal.Zero
		}
		for _, c := range b.Categories {
			catIDs = append(catIDs, c.ID)
		}
	}
	var tagIDs []int64
	for _, t := range b.Tags {
		tagIDs = append(tagIDs, t.ID)
	}
	total, err := filter.BudgetSpent(ctx, s.DB, start.Format("2006-01-02"), end.Format("2006-01-02"), catIDs, tagIDs)
	if err != nil {
		return decimal.Zero
	}
	return total.DivRound(targetRate, rateDivPrecision)
}

// resolveCurrency returns the budget currency (base when unset) and its decimals.
func (s Budgets) resolveCurrency(ctx context.Context, id *int64) (int64, int, error) {
	curs := Currencies{DB: s.DB}
	var c *Currency
	var err error
	if id != nil {
		c, err = curs.ByID(ctx, *id)
	} else {
		c, err = curs.Base(ctx)
	}
	if err != nil {
		return 0, 0, err
	}
	if c == nil {
		return 0, 0, fmt.Errorf("currency required")
	}
	return c.ID, c.Decimals, nil
}

func budgetPeriodDates(b Budget) (time.Time, time.Time) {
	now := time.Now()
	anchor := now
	if b.StartDate != nil && *b.StartDate != "" {
		if t, err := time.ParseInLocation("2006-01-02", *b.StartDate, now.Location()); err == nil {
			anchor = t
		}
	}
	switch b.Period {
	case "one_time":
		start := startOfMonth(now)
		end := endOfMonth(now)
		if b.StartDate != nil && *b.StartDate != "" {
			if t, err := time.ParseInLocation("2006-01-02", *b.StartDate, now.Location()); err == nil {
				start = t
			}
		}
		if b.EndDate != nil && *b.EndDate != "" {
			if t, err := time.ParseInLocation("2006-01-02", *b.EndDate, now.Location()); err == nil {
				end = t
			}
		}
		return start, end
	case "weekly":
		return startOfWeek(anchor), endOfWeek(anchor)
	case "yearly":
		return time.Date(anchor.Year(), 1, 1, 0, 0, 0, 0, anchor.Location()),
			time.Date(anchor.Year(), 12, 31, 0, 0, 0, 0, anchor.Location())
	default: // monthly
		return startOfMonth(anchor), endOfMonth(anchor)
	}
}

func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func endOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location())
}

func startOfWeek(t time.Time) time.Time {
	// Carbon default: Monday.
	offset := (int(t.Weekday()) + 6) % 7
	return time.Date(t.Year(), t.Month(), t.Day()-offset, 0, 0, 0, 0, t.Location())
}

func endOfWeek(t time.Time) time.Time {
	return startOfWeek(t).AddDate(0, 0, 6)
}

func (s Budgets) list(ctx context.Context, id sql.NullInt64) ([]Budget, error) {
	rows, err := db.Q(s.DB).ListBudgets(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Budget, 0, len(rows))
	for _, r := range rows {
		out = append(out, budgetFromRow(r))
	}
	curs := Currencies{DB: s.DB}
	for i := range out {
		if out[i].CurrencyID != nil {
			if c, _ := curs.ByID(ctx, *out[i].CurrencyID); c != nil {
				out[i].Currency = c
			}
		}
		out[i].Categories, _ = s.cats(ctx, out[i].ID)
		out[i].Tags, _ = s.tags(ctx, out[i].ID)
		p := s.CalculateProgress(ctx, &out[i])
		out[i].Progress = &p
	}
	return out, nil
}

func (s Budgets) cats(ctx context.Context, id int64) ([]Category, error) {
	rows, err := db.Q(s.DB).ListBudgetCategories(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Category, 0, len(rows))
	for _, r := range rows {
		out = append(out, categoryFrom(r.ID, r.Name, r.Type, r.Icon, r.Color, r.IsDefault, r.TransactionsCount))
	}
	return out, nil
}

func (s Budgets) tags(ctx context.Context, id int64) ([]Tag, error) {
	rows, err := db.Q(s.DB).ListBudgetTags(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Tag, 0, len(rows))
	for _, r := range rows {
		out = append(out, tagFromList(r.ID, r.Name, r.CreatedAt, r.TransactionsCount))
	}
	return out, nil
}

func (s Budgets) saveCats(ctx context.Context, id int64, ids []int64) error {
	if err := db.Q(s.DB).DeleteBudgetCategories(ctx, id); err != nil {
		return err
	}
	for _, catID := range ids {
		if err := db.Q(s.DB).InsertBudgetCategory(ctx, sqlc.InsertBudgetCategoryParams{BudgetID: id, CategoryID: catID}); err != nil {
			return err
		}
	}
	return nil
}

func (s Budgets) saveTags(ctx context.Context, id int64, ids []int64) error {
	if err := db.Q(s.DB).DeleteBudgetTags(ctx, id); err != nil {
		return err
	}
	for _, tagID := range ids {
		if err := db.Q(s.DB).InsertBudgetTag(ctx, sqlc.InsertBudgetTagParams{BudgetID: id, TagID: tagID}); err != nil {
			return err
		}
	}
	return nil
}

func budgetFromRow(r sqlc.ListBudgetsRow) Budget {
	b := Budget{ID: r.ID, Name: r.Name, Amount: money.FromMinor(r.Amount, int(r.Decimals)), Period: r.Period, IsGlobal: r.IsGlobal != 0, IsActive: r.IsActive != 0}
	currencyID := r.CurrencyID
	b.CurrencyID = &currencyID
	if r.StartDate.Valid {
		b.StartDate = &r.StartDate.String
	}
	if r.EndDate.Valid {
		b.EndDate = &r.EndDate.String
	}
	if r.NotifyAtPercent.Valid {
		v := int(r.NotifyAtPercent.Int64)
		b.NotifyAtPercent = &v
	}
	return b
}
