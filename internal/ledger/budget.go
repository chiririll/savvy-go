package ledger

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/shopspring/decimal"

	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/money"
	"savvy-go/internal/platform/calendar"
	"savvy-go/internal/store"
)

// BudgetProgress is how much of a budget is used; the amounts are in the
// budget's currency.
type BudgetProgress struct {
	Spent       money.Money
	Remaining   money.Money
	Percent     float64
	PeriodStart string
	PeriodEnd   string
	IsExceeded  bool
}

type Budget struct {
	ID              int64
	Name            string
	Amount          money.Money
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

type Budgets struct{ DB store.DB }

func (s Budgets) repo() repo.Repo { return repo.Repo{DB: s.DB} }

func (s Budgets) All(ctx context.Context) ([]Budget, error) {
	return s.list(ctx, nil)
}

func (s Budgets) ByID(ctx context.Context, id int64) (*Budget, error) {
	list, err := s.list(ctx, &id)
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
	unit, err := s.resolveCurrency(ctx, in.CurrencyID)
	if err != nil {
		return nil, err
	}
	amount, err := money.FromInput(in.Amount, unit)
	if err != nil {
		return nil, err
	}
	id, err := s.repo().InsertBudget(ctx, repo.BudgetFields{
		Name: in.Name, Amount: amount, Period: in.Period, StartDate: in.StartDate, EndDate: in.EndDate,
		IsGlobal: global, NotifyAtPercent: in.NotifyAtPercent, IsActive: active,
	}, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if err := s.repo().SetBudgetCategories(ctx, id, in.CategoryIDs); err != nil {
		return nil, err
	}
	if err := s.repo().SetBudgetTags(ctx, id, in.TagIDs); err != nil {
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
		in.Amount = cur.Amount.Decimal()
	}
	if in.Period == "" {
		in.Period = cur.Period
	}
	global := cur.IsGlobal
	if in.IsGlobal != nil {
		global = *in.IsGlobal
	}
	active := cur.IsActive
	if in.IsActive != nil {
		active = *in.IsActive
	}
	unit, err := s.resolveCurrency(ctx, in.CurrencyID)
	if err != nil {
		return nil, err
	}
	amount, err := money.FromInput(in.Amount, unit)
	if err != nil {
		return nil, err
	}
	err = s.repo().UpdateBudget(ctx, id, repo.BudgetFields{
		Name: in.Name, Amount: amount, Period: in.Period, StartDate: in.StartDate, EndDate: in.EndDate,
		IsGlobal: global, NotifyAtPercent: in.NotifyAtPercent, IsActive: active,
	}, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if in.HasCategoryIDs {
		if err := s.repo().SetBudgetCategories(ctx, id, in.CategoryIDs); err != nil {
			return nil, err
		}
	}
	if in.HasTagIDs {
		if err := s.repo().SetBudgetTags(ctx, id, in.TagIDs); err != nil {
			return nil, err
		}
	}
	return s.ByID(ctx, id)
}

func (s Budgets) Delete(ctx context.Context, id int64) error {
	return s.repo().DeleteBudget(ctx, id)
}

func (s Budgets) CalculateProgress(ctx context.Context, b *Budget) BudgetProgress {
	start, end := budgetPeriodDates(*b)
	spent := s.spent(ctx, b, start, end)
	remaining := money.Max(money.Zero(b.Amount.Unit()), b.Amount.Sub(spent))
	percent := 0.0
	if b.Amount.IsPositive() {
		percent = math.Round(spent.Decimal().Div(b.Amount.Decimal()).InexactFloat64()*1000) / 10
	}
	return BudgetProgress{
		Spent: spent, Remaining: remaining, Percent: percent,
		PeriodStart: start.Format("2006-01-02"), PeriodEnd: end.Format("2006-01-02"),
		IsExceeded: spent.Cmp(b.Amount) > 0,
	}
}

// spent is what the budget's scope spent in [start, end], in its currency.
func (s Budgets) spent(ctx context.Context, b *Budget, start, end time.Time) money.Money {
	unit := b.Amount.Unit()
	targetRate := one
	if b.Currency != nil && b.Currency.Rate.IsPositive() {
		targetRate = b.Currency.Rate
	}

	var catIDs []int64
	if !b.IsGlobal {
		if len(b.Categories) == 0 {
			return money.Zero(unit)
		}
		for _, c := range b.Categories {
			catIDs = append(catIDs, c.ID)
		}
	}
	var tagIDs []int64
	for _, t := range b.Tags {
		tagIDs = append(tagIDs, t.ID)
	}
	spent, err := s.repo().BudgetSpent(ctx, start.Format("2006-01-02"), end.Format("2006-01-02"), catIDs, tagIDs, unit, targetRate)
	if err != nil {
		return money.Zero(unit)
	}
	return spent
}

// resolveCurrency returns the unit of the budget currency (base when unset).
func (s Budgets) resolveCurrency(ctx context.Context, id *int64) (money.Unit, error) {
	curs := Currencies{DB: s.DB}
	var c *Currency
	var err error
	if id != nil {
		c, err = curs.ByID(ctx, *id)
	} else {
		c, err = curs.Base(ctx)
	}
	if err != nil {
		return money.Unit{}, err
	}
	if c == nil {
		return money.Unit{}, fmt.Errorf("currency required")
	}
	return c.Unit(), nil
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
		start := calendar.StartOfMonth(now)
		end := calendar.EndOfMonth(now)
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
		return calendar.StartOfWeek(anchor), calendar.EndOfWeek(anchor)
	case "yearly":
		return time.Date(anchor.Year(), 1, 1, 0, 0, 0, 0, anchor.Location()),
			time.Date(anchor.Year(), 12, 31, 0, 0, 0, 0, anchor.Location())
	default: // monthly
		return calendar.StartOfMonth(anchor), calendar.EndOfMonth(anchor)
	}
}

func (s Budgets) list(ctx context.Context, id *int64) ([]Budget, error) {
	rows, err := s.repo().Budgets(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Budget, len(rows))
	curs := Currencies{DB: s.DB}
	for i, r := range rows {
		currencyID := r.Amount.Unit().ID
		out[i] = Budget{
			ID: r.ID, Name: r.Name, Amount: r.Amount, CurrencyID: &currencyID, Period: r.Period,
			StartDate: r.StartDate, EndDate: r.EndDate, IsGlobal: r.IsGlobal, NotifyAtPercent: r.NotifyAtPercent,
			IsActive: r.IsActive,
		}
		if c, _ := curs.ByID(ctx, currencyID); c != nil {
			out[i].Currency = c
		}
		out[i].Categories, _ = s.cats(ctx, r.ID)
		out[i].Tags, _ = s.tags(ctx, r.ID)
		p := s.CalculateProgress(ctx, &out[i])
		out[i].Progress = &p
	}
	return out, nil
}

func (s Budgets) cats(ctx context.Context, id int64) ([]Category, error) {
	rows, err := s.repo().BudgetCategories(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Category, len(rows))
	for i, r := range rows {
		out[i] = categoryFrom(r)
	}
	return out, nil
}

func (s Budgets) tags(ctx context.Context, id int64) ([]Tag, error) {
	rows, err := s.repo().BudgetTags(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Tag, len(rows))
	for i, r := range rows {
		out[i] = tagFrom(r)
	}
	return out, nil
}
