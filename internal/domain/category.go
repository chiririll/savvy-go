package domain

import (
	"context"
	"database/sql"
	"fmt"
	"time"
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
		"id":   c.ID,
		"name": c.Name,
		"type": c.Type,
		"icon": c.Icon,
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
	q := `SELECT c.id, c.name, c.type, c.icon, c.color,
		(SELECT COUNT(*) FROM transactions t WHERE t.category_id = c.id) 
		FROM categories c`
	var args []any
	if typ != "" {
		q += ` WHERE c.type = ?`
		args = append(args, typ)
	}
	q += ` ORDER BY c.name`
	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Category
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s Categories) ByID(ctx context.Context, id int64) (*Category, error) {
	c, err := scanCategory(s.DB.QueryRowContext(ctx, `
		SELECT id, name, type, icon, color,
			(SELECT COUNT(*) FROM transactions t WHERE t.category_id = categories.id)
		FROM categories WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (s Categories) Create(ctx context.Context, c Category) (*Category, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.DB.ExecContext(ctx,
		`INSERT INTO categories (name, type, icon, color, created_at, updated_at) VALUES (?,?,?,?,?,?)`,
		c.Name, c.Type, c.Icon, c.Color, now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

func (s Categories) Update(ctx context.Context, id int64, c Category) (*Category, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.ExecContext(ctx,
		`UPDATE categories SET name=?, type=?, icon=?, color=?, updated_at=? WHERE id=?`,
		c.Name, c.Type, c.Icon, c.Color, now, id)
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
	var n int
	_ = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM categories WHERE type = ?`, c.Type).Scan(&n)
	if n <= 1 {
		return fmt.Errorf("last")
	}
	_, err = s.DB.ExecContext(ctx, `DELETE FROM categories WHERE id = ?`, id)
	return err
}

func (s Categories) Statistics(ctx context.Context, id int64, start, end string) (map[string]any, error) {
	c, err := s.ByID(ctx, id)
	if err != nil || c == nil {
		return nil, err
	}
	q := `SELECT COUNT(*), COALESCE(SUM(amount),0) FROM transactions WHERE category_id = ? AND status = 'confirmed'`
	args := []any{id}
	if start != "" {
		q += ` AND date >= ?`
		args = append(args, start)
	}
	if end != "" {
		q += ` AND date <= ?`
		args = append(args, end)
	}
	var count int
	var total float64
	_ = s.DB.QueryRowContext(ctx, q, args...).Scan(&count, &total)
	return map[string]any{
		"category_id":         c.ID,
		"category_name":       c.Name,
		"type":                c.Type,
		"transactions_count":  count,
		"total_amount":        total,
	}, nil
}

func scanCategory(row interface{ Scan(...any) error }) (Category, error) {
	var c Category
	var icon, color sql.NullString
	err := row.Scan(&c.ID, &c.Name, &c.Type, &icon, &color, &c.TransactionsCount)
	if icon.Valid {
		c.Icon = &icon.String
	}
	if color.Valid {
		c.Color = &color.String
	}
	return c, err
}
