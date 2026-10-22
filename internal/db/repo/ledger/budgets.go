package ledger

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/money"
)

type Budget struct {
	ID                 int64
	Name, Period       string
	Amount             money.Money
	StartDate, EndDate *string
	IsGlobal, IsActive bool
	NotifyAtPercent    *int
}

type BudgetFields struct {
	Name, Period       string
	Amount             money.Money
	StartDate, EndDate *string
	IsGlobal, IsActive bool
	NotifyAtPercent    *int
}

// Budgets lists budgets, or the one with id when id is not nil.
func (r Repo) Budgets(ctx context.Context, id *int64) ([]Budget, error) {
	rows, err := r.q().ListBudgets(ctx, db.NullInt64(id))
	if err != nil {
		return nil, err
	}
	out := make([]Budget, len(rows))
	for i, x := range rows {
		out[i] = Budget{
			ID: x.ID, Name: x.Name, Period: x.Period,
			Amount: money.New(x.Amount, money.Unit{ID: x.CurrencyID, Decimals: int(x.Decimals)}),
			StartDate: optString(
				x.StartDate,
			), EndDate: optString(x.EndDate), IsGlobal: x.IsGlobal != 0, IsActive: x.IsActive != 0,
			NotifyAtPercent: optInt(x.NotifyAtPercent),
		}
	}
	return out, nil
}

func (r Repo) InsertBudget(ctx context.Context, b BudgetFields, now time.Time) (int64, error) {
	res, err := r.q().InsertBudget(ctx, sqlc.InsertBudgetParams{
		Name: b.Name, Amount: b.Amount.Minor(), CurrencyID: b.Amount.Unit().ID, Period: b.Period,
		StartDate: db.NullString(b.StartDate), EndDate: db.NullString(b.EndDate),
		IsGlobal: db.BoolInt(b.IsGlobal), NotifyAtPercent: db.NullInt(b.NotifyAtPercent),
		IsActive: db.BoolInt(b.IsActive), CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r Repo) UpdateBudget(ctx context.Context, id int64, b BudgetFields, now time.Time) error {
	return r.q().UpdateBudget(ctx, sqlc.UpdateBudgetParams{
		Name: b.Name, Amount: b.Amount.Minor(), CurrencyID: b.Amount.Unit().ID, Period: b.Period,
		StartDate: db.NullString(b.StartDate), EndDate: db.NullString(b.EndDate),
		IsGlobal: db.BoolInt(b.IsGlobal), NotifyAtPercent: db.NullInt(b.NotifyAtPercent),
		IsActive: db.BoolInt(b.IsActive), UpdatedAt: db.TimeNS(now), ID: id,
	})
}

func (r Repo) DeleteBudget(ctx context.Context, id int64) error { return r.q().DeleteBudget(ctx, id) }

func (r Repo) BudgetCategories(ctx context.Context, id int64) ([]Category, error) {
	rows, err := r.q().ListBudgetCategories(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Category, len(rows))
	for i, x := range rows {
		out[i] = categoryOf(x.ID, x.Name, x.Type, x.Icon, x.Color, x.IsDefault, x.TransactionsCount)
	}
	return out, nil
}

func (r Repo) BudgetTags(ctx context.Context, id int64) ([]Tag, error) {
	rows, err := r.q().ListBudgetTags(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Tag, len(rows))
	for i, x := range rows {
		out[i] = tagOf(x.ID, x.Name, x.CreatedAt, x.TransactionsCount)
	}
	return out, nil
}

// SetBudgetCategories replaces the categories a budget covers.
func (r Repo) SetBudgetCategories(ctx context.Context, id int64, categoryIDs []int64) error {
	if err := r.q().DeleteBudgetCategories(ctx, id); err != nil {
		return err
	}
	for _, c := range categoryIDs {
		if err := r.q().InsertBudgetCategory(ctx, sqlc.InsertBudgetCategoryParams{BudgetID: id, CategoryID: c}); err != nil {
			return err
		}
	}
	return nil
}

// SetBudgetTags replaces the tags a budget covers.
func (r Repo) SetBudgetTags(ctx context.Context, id int64, tagIDs []int64) error {
	if err := r.q().DeleteBudgetTags(ctx, id); err != nil {
		return err
	}
	for _, t := range tagIDs {
		if err := r.q().InsertBudgetTag(ctx, sqlc.InsertBudgetTagParams{BudgetID: id, TagID: t}); err != nil {
			return err
		}
	}
	return nil
}

// MonthlyBudget is a monthly limit with what is needed to fold it into the
// base currency.
type MonthlyBudget struct {
	Amount money.Money
	Rate   decimal.Decimal
	IsBase bool
}

// GlobalMonthlyBudget returns the global monthly budget; nil when there is none.
func (r Repo) GlobalMonthlyBudget(ctx context.Context) *MonthlyBudget {
	x, err := r.q().GetGlobalMonthlyBudget(ctx)
	if err != nil {
		return nil
	}
	return &MonthlyBudget{
		Amount: money.New(x.Amount, money.Unit{ID: x.CurrencyID, Decimals: int(x.Decimals)}),
		Rate:   x.Rate,
		IsBase: x.IsBase != 0,
	}
}

// MonthlyBudgets lists the active monthly budgets.
func (r Repo) MonthlyBudgets(ctx context.Context) ([]MonthlyBudget, error) {
	rows, err := r.q().ListMonthlyBudgets(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]MonthlyBudget, len(rows))
	for i, x := range rows {
		out[i] = MonthlyBudget{
			Amount: money.New(x.Amount, money.Unit{ID: x.CurrencyID, Decimals: int(x.Decimals)}),
			Rate:   x.Rate,
			IsBase: x.IsBase != 0,
		}
	}
	return out, nil
}
