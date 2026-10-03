package domain

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/money"
)

type Category struct {
	ID                int64
	Name              string
	Type              string
	Icon              *string
	Color             *string
	IsDefault         bool
	TransactionsCount int
	TotalAmount       *decimal.Decimal
}

type Categories struct{ DB *sql.DB }

func (s Categories) All(ctx context.Context, typ string) ([]Category, error) {
	rows, err := db.Q(s.DB).ListCategories(ctx, db.Narg(typ))
	if err != nil {
		return nil, err
	}
	out := make([]Category, 0, len(rows))
	for _, r := range rows {
		out = append(out, categoryFrom(r.ID, r.Name, r.Type, r.Icon, r.Color, r.IsDefault, r.Count))
	}
	return out, nil
}

func (s Categories) ByID(ctx context.Context, id int64) (*Category, error) {
	r, err := db.Q(s.DB).GetCategory(ctx, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c := categoryFrom(r.ID, r.Name, r.Type, r.Icon, r.Color, r.IsDefault, r.Count)
	return &c, nil
}

func (s Categories) Create(ctx context.Context, c Category) (*Category, error) {
	if c.IsDefault {
		if err := db.Q(s.DB).ClearDefaultCategory(ctx, c.Type); err != nil {
			return nil, err
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Q(s.DB).InsertCategory(ctx, sqlc.InsertCategoryParams{
		Name: c.Name, Type: c.Type, Icon: db.NullString(c.Icon), Color: db.NullString(c.Color),
		IsDefault: db.BoolInt(c.IsDefault), CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
	})
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
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
		if err := db.Q(s.DB).ClearDefaultCategory(ctx, c.Type); err != nil {
			return nil, err
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err = db.Q(s.DB).UpdateCategory(ctx, sqlc.UpdateCategoryParams{
		Name: c.Name, Type: c.Type, Icon: db.NullString(c.Icon), Color: db.NullString(c.Color),
		IsDefault: db.BoolInt(c.IsDefault), UpdatedAt: db.NS(now), ID: id,
	})
	if err != nil {
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
		tx, err := s.DB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback() }()
		q := db.Q(s.DB).WithTx(tx)
		if err := q.ReassignCategoryTransactions(ctx, sqlc.ReassignCategoryTransactionsParams{
			CategoryID: db.NI(*successorID), CategoryID_2: db.NI(id),
		}); err != nil {
			return err
		}
		if err := q.DeleteCategory(ctx, id); err != nil {
			return err
		}
		return tx.Commit()
	}
	n, _ := db.Q(s.DB).CountCategoriesByType(ctx, c.Type)
	if n <= 1 {
		return fmt.Errorf("last")
	}
	return db.Q(s.DB).DeleteCategory(ctx, id)
}

func (s Categories) SetDefault(ctx context.Context, id int64) (*Category, error) {
	c, err := s.ByID(ctx, id)
	if err != nil || c == nil {
		return c, err
	}
	if c.IsDefault {
		return c, nil
	}
	if err := db.Q(s.DB).ClearDefaultCategory(ctx, c.Type); err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if err := db.Q(s.DB).SetCategoryDefault(ctx, sqlc.SetCategoryDefaultParams{UpdatedAt: db.NS(now), ID: id}); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Categories) Statistics(ctx context.Context, id int64, start, end string) (map[string]any, error) {
	c, err := s.ByID(ctx, id)
	if err != nil || c == nil {
		return nil, err
	}
	rows, _ := db.Q(s.DB).CategoryStatistics(ctx, sqlc.CategoryStatisticsParams{
		CategoryID: db.NI(id), StartDate: db.Narg(start), EndDate: db.Narg(end),
	})
	count := int64(0)
	total := decimal.Zero
	for _, row := range rows {
		count += row.Cnt
		total = total.Add(money.FromMinor(row.Total, int(row.Decimals)).Mul(row.Rate))
	}
	baseDec := 2
	if base, _ := (Currencies{DB: s.DB}).Base(ctx); base != nil {
		baseDec = base.Decimals
	}
	return map[string]any{
		"category_id":        c.ID,
		"category_name":      c.Name,
		"type":               c.Type,
		"transactions_count": int(count),
		"total_amount":       money.Number(total, baseDec),
	}, nil
}

func categoryFrom(id int64, name, typ string, icon, color sql.NullString, isDefault, count int64) Category {
	c := Category{ID: id, Name: name, Type: typ, IsDefault: isDefault != 0, TransactionsCount: int(count)}
	if icon.Valid {
		c.Icon = &icon.String
	}
	if color.Valid {
		c.Color = &color.String
	}
	return c
}

func asFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	}
	return 0
}
