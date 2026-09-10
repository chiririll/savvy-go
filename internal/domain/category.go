package domain

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
)

type Category struct {
	ID                int64
	Name              string
	Type              string
	Icon              *string
	Color             *string
	TransactionsCount int
	TotalAmount       *float64
}

func (c Category) JSON() map[string]any {
	m := map[string]any{
		"id":    c.ID,
		"name":  c.Name,
		"type":  c.Type,
		"icon":  c.Icon,
		"color": c.Color,
	}
	if c.TransactionsCount > 0 || true {
		m["transactionsCount"] = c.TransactionsCount
	}
	if c.TotalAmount != nil {
		m["totalAmount"] = *c.TotalAmount
	}
	return m
}

type Categories struct{ DB *sql.DB }

func (s Categories) All(ctx context.Context, typ string) ([]Category, error) {
	rows, err := db.Q(s.DB).ListCategories(ctx, db.Narg(typ))
	if err != nil {
		return nil, err
	}
	out := make([]Category, 0, len(rows))
	for _, r := range rows {
		out = append(out, categoryFrom(r.ID, r.Name, r.Type, r.Icon, r.Color, r.Count))
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
	c := categoryFrom(r.ID, r.Name, r.Type, r.Icon, r.Color, r.Count)
	return &c, nil
}

func (s Categories) Create(ctx context.Context, c Category) (*Category, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Q(s.DB).InsertCategory(ctx, sqlc.InsertCategoryParams{
		Name: c.Name, Type: c.Type, Icon: db.NullString(c.Icon), Color: db.NullString(c.Color),
		CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
	})
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

func (s Categories) Update(ctx context.Context, id int64, c Category) (*Category, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	err := db.Q(s.DB).UpdateCategory(ctx, sqlc.UpdateCategoryParams{
		Name: c.Name, Type: c.Type, Icon: db.NullString(c.Icon), Color: db.NullString(c.Color),
		UpdatedAt: db.NS(now), ID: id,
	})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Categories) Delete(ctx context.Context, id int64) error {
	c, err := s.ByID(ctx, id)
	if err != nil || c == nil {
		return err
	}
	if c.TransactionsCount > 0 {
		return fmt.Errorf("has transactions")
	}
	n, _ := db.Q(s.DB).CountCategoriesByType(ctx, c.Type)
	if n <= 1 {
		return fmt.Errorf("last")
	}
	return db.Q(s.DB).DeleteCategory(ctx, id)
}

func (s Categories) Statistics(ctx context.Context, id int64, start, end string) (map[string]any, error) {
	c, err := s.ByID(ctx, id)
	if err != nil || c == nil {
		return nil, err
	}
	row, _ := db.Q(s.DB).CategoryStatistics(ctx, sqlc.CategoryStatisticsParams{
		CategoryID: db.NI(id), StartDate: db.Narg(start), EndDate: db.Narg(end),
	})
	return map[string]any{
		"category_id":        c.ID,
		"category_name":      c.Name,
		"type":               c.Type,
		"transactions_count": int(row.Count),
		"total_amount":       asFloat64(row.Coalesce),
	}, nil
}

func categoryFrom(id int64, name, typ string, icon, color sql.NullString, count int64) Category {
	c := Category{ID: id, Name: name, Type: typ, TransactionsCount: int(count)}
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
