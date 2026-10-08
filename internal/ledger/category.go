package ledger

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/money"
	"savvy-go/internal/store"
)

type Category struct {
	ID                int64
	Name              string
	Type              string
	Icon              *string
	Color             *string
	IsDefault         bool
	TransactionsCount int
	TotalAmount       *money.Money
}

type Categories struct{ DB store.DB }

func (s Categories) repo() repo.Repo { return repo.Repo{DB: s.DB} }

func (s Categories) All(ctx context.Context, typ string) ([]Category, error) {
	rows, err := s.repo().Categories(ctx, typ)
	if err != nil {
		return nil, err
	}
	out := make([]Category, len(rows))
	for i, r := range rows {
		out[i] = categoryFrom(r)
	}
	return out, nil
}

func (s Categories) ByID(ctx context.Context, id int64) (*Category, error) {
	r, err := s.repo().Category(ctx, id)
	if r == nil || err != nil {
		return nil, err
	}
	c := categoryFrom(*r)
	return &c, nil
}

func (s Categories) Create(ctx context.Context, c Category) (*Category, error) {
	if c.IsDefault {
		if err := s.repo().ClearDefaultCategory(ctx, c.Type); err != nil {
			return nil, err
		}
	}
	id, err := s.repo().InsertCategory(ctx, categoryRow(c), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Categories) Update(ctx context.Context, id int64, c Category) (*Category, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if cur.IsDefault && c.Type != cur.Type {
		return nil, fmt.Errorf("cannot change type of default category")
	}
	if cur.IsDefault && !c.IsDefault {
		return nil, fmt.Errorf("cannot unset default")
	}
	if c.IsDefault && !cur.IsDefault {
		if err := s.repo().ClearDefaultCategory(ctx, c.Type); err != nil {
			return nil, err
		}
	}
	if err := s.repo().UpdateCategory(ctx, id, categoryRow(c), time.Now().UTC()); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Categories) Delete(ctx context.Context, id int64, successorID *int64) error {
	c, err := s.ByID(ctx, id)
	if err != nil || c == nil {
		return err
	}
	if c.IsDefault {
		return fmt.Errorf("default")
	}
	if c.TransactionsCount > 0 {
		if successorID == nil {
			return fmt.Errorf("has transactions")
		}
		successor, err := s.ByID(ctx, *successorID)
		if err != nil {
			return err
		}
		if successor == nil || successor.Type != c.Type || successor.ID == c.ID {
			return fmt.Errorf("invalid successor")
		}
		return s.repo().ReplaceCategory(ctx, id, *successorID)
	}
	if s.repo().CountCategoriesOfType(ctx, c.Type) <= 1 {
		return fmt.Errorf("last")
	}
	return s.repo().DeleteCategory(ctx, id)
}

func (s Categories) SetDefault(ctx context.Context, id int64) (*Category, error) {
	c, err := s.ByID(ctx, id)
	if err != nil || c == nil {
		return c, err
	}
	if c.IsDefault {
		return c, nil
	}
	if err := s.repo().ClearDefaultCategory(ctx, c.Type); err != nil {
		return nil, err
	}
	if err := s.repo().SetCategoryDefault(ctx, id, time.Now().UTC()); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

// CategoryStatistics totals a category's transactions in the base currency,
// rounded to its decimals.
type CategoryStatistics struct {
	Category Category
	Count    int
	Total    money.Money
}

func (s Categories) Statistics(ctx context.Context, id int64, start, end string) (*CategoryStatistics, error) {
	c, err := s.ByID(ctx, id)
	if err != nil || c == nil {
		return nil, err
	}
	out := &CategoryStatistics{Category: *c}
	total := decimal.Zero
	for _, row := range s.repo().CategoryStatistics(ctx, id, start, end) {
		out.Count += row.Count
		total = total.Add(decimal.New(row.Minor, -int32(row.Decimals)).Mul(row.Rate))
	}
	var unit money.Unit // no base currency means nothing was counted either
	if base, _ := (Currencies{DB: s.DB}).Base(ctx); base != nil {
		unit = base.Unit()
	}
	out.Total = money.FromDecimal(total, unit)
	return out, nil
}

func categoryFrom(r repo.Category) Category {
	return Category{
		ID: r.ID, Name: r.Name, Type: r.Type, Icon: r.Icon, Color: r.Color,
		IsDefault: r.IsDefault, TransactionsCount: r.TransactionsCount,
	}
}

func categoryRow(c Category) repo.Category {
	return repo.Category{Name: c.Name, Type: c.Type, Icon: c.Icon, Color: c.Color, IsDefault: c.IsDefault}
}
