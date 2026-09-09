package domain

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"
)

type Currency struct {
	ID       int64
	Code     string
	Name     string
	Symbol   string
	Decimals int
	IsBase   bool
	Rate     float64
}

func (c Currency) JSON() map[string]any {
	return map[string]any{
		"id":       c.ID,
		"code":     c.Code,
		"name":     c.Name,
		"symbol":   c.Symbol,
		"decimals": c.Decimals,
		"isBase":   c.IsBase,
		"rate":     c.Rate,
	}
}

func (c Currency) ConvertToBase(amount float64) float64 {
	if c.IsBase {
		return amount
	}
	return amount * c.Rate
}

func (c Currency) ConvertFromBase(amount float64) float64 {
	if c.IsBase || c.Rate == 0 {
		return amount
	}
	return amount / c.Rate
}

func Convert(amount float64, from, to Currency) float64 {
	if from.ID == to.ID {
		return amount
	}
	return to.ConvertFromBase(from.ConvertToBase(amount))
}

type Currencies struct{ DB *sql.DB }

func (s Currencies) All(ctx context.Context) ([]Currency, error) {
	return s.query(ctx, `SELECT id, code, name, symbol, decimals, is_base, rate FROM currencies ORDER BY code`)
}

func (s Currencies) ByID(ctx context.Context, id int64) (*Currency, error) {
	return scanCurrency(s.DB.QueryRowContext(ctx,
		`SELECT id, code, name, symbol, decimals, is_base, rate FROM currencies WHERE id = ?`, id))
}

func (s Currencies) ByCode(ctx context.Context, code string) (*Currency, error) {
	return scanCurrency(s.DB.QueryRowContext(ctx,
		`SELECT id, code, name, symbol, decimals, is_base, rate FROM currencies WHERE code = ?`, strings.ToUpper(code)))
}

func (s Currencies) Base(ctx context.Context) (*Currency, error) {
	return scanCurrency(s.DB.QueryRowContext(ctx,
		`SELECT id, code, name, symbol, decimals, is_base, rate FROM currencies WHERE is_base = 1 LIMIT 1`))
}

func (s Currencies) Create(ctx context.Context, c Currency) (*Currency, error) {
	c.Code = strings.ToUpper(strings.TrimSpace(c.Code))
	if c.Decimals < 0 {
		c.Decimals = 2
	}
	if c.Rate == 0 {
		c.Rate = 1
	}
	if c.IsBase {
		if _, err := s.DB.ExecContext(ctx, `UPDATE currencies SET is_base = 0 WHERE is_base = 1`); err != nil {
			return nil, err
		}
		c.Rate = 1
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.DB.ExecContext(ctx, `
		INSERT INTO currencies (code, name, symbol, decimals, is_base, rate, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.Code, c.Name, c.Symbol, c.Decimals, boolInt(c.IsBase), c.Rate, now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

func (s Currencies) Update(ctx context.Context, id int64, c Currency) (*Currency, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if cur.IsBase && !c.IsBase {
		return nil, fmt.Errorf("cannot unset base")
	}
	if cur.IsBase && c.Rate != 1 {
		return nil, fmt.Errorf("base rate")
	}
	if c.IsBase && !cur.IsBase {
		if _, err := s.DB.ExecContext(ctx, `UPDATE currencies SET is_base = 0 WHERE is_base = 1`); err != nil {
			return nil, err
		}
		c.Rate = 1
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.DB.ExecContext(ctx, `
		UPDATE currencies SET name=?, symbol=?, decimals=?, is_base=?, rate=?, updated_at=? WHERE id=?`,
		c.Name, c.Symbol, c.Decimals, boolInt(c.IsBase), c.Rate, now, id)
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Currencies) Delete(ctx context.Context, id int64) error {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return err
	}
	var used int
	_ = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM accounts WHERE currency_id = ?`, id).Scan(&used)
	if used > 0 {
		return fmt.Errorf("in use")
	}
	if cur.IsBase {
		return fmt.Errorf("base")
	}
	var n int
	_ = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM currencies`).Scan(&n)
	if n <= 1 {
		return fmt.Errorf("last")
	}
	_, err = s.DB.ExecContext(ctx, `DELETE FROM currencies WHERE id = ?`, id)
	return err
}

func (s Currencies) SetBase(ctx context.Context, id int64) (*Currency, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if cur.IsBase {
		return cur, nil
	}
	newRate := cur.Rate
	if newRate == 0 {
		newRate = 1
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id, rate FROM currencies WHERE id != ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var oid int64
		var rate float64
		if err := rows.Scan(&oid, &rate); err != nil {
			return nil, err
		}
		if _, err := s.DB.ExecContext(ctx, `UPDATE currencies SET rate = ?, is_base = 0 WHERE id = ?`, rate/newRate, oid); err != nil {
			return nil, err
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.DB.ExecContext(ctx, `UPDATE currencies SET is_base = 1, rate = 1, updated_at = ? WHERE id = ?`, now, id)
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Currencies) FindOrCreateByCode(ctx context.Context, code string) (*Currency, error) {
	if c, err := s.ByCode(ctx, code); err != nil || c != nil {
		return c, err
	}
	item := catalogItem(code)
	if item == nil {
		return nil, fmt.Errorf("unknown code")
	}
	var n int
	_ = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM currencies`).Scan(&n)
	item.IsBase = n == 0
	if item.IsBase {
		item.Rate = 1
	}
	return s.Create(ctx, *item)
}

func (s Currencies) Catalog(ctx context.Context) []map[string]any {
	existing := map[string]bool{}
	rows, err := s.DB.QueryContext(ctx, `SELECT code FROM currencies`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var code string
			_ = rows.Scan(&code)
			existing[strings.ToUpper(code)] = true
		}
	}
	var out []map[string]any
	for _, item := range builtinCatalog {
		if existing[item.Code] {
			continue
		}
		out = append(out, map[string]any{
			"code": item.Code, "name": item.Name, "symbol": item.Symbol,
			"decimals": item.Decimals, "rate": item.Rate,
		})
	}
	return out
}

func (s Currencies) query(ctx context.Context, q string, args ...any) ([]Currency, error) {
	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Currency
	for rows.Next() {
		c, err := scanCurrencyRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func scanCurrency(row interface{ Scan(...any) error }) (*Currency, error) {
	c, err := scanCurrencyRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func scanCurrencyRow(row interface{ Scan(...any) error }) (*Currency, error) {
	var c Currency
	var base int
	if err := row.Scan(&c.ID, &c.Code, &c.Name, &c.Symbol, &c.Decimals, &base, &c.Rate); err != nil {
		return nil, err
	}
	c.IsBase = base != 0
	return &c, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func roundTo(v float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(v*p) / p
}

var builtinCatalog = []Currency{
	{Code: "USD", Name: "US Dollar", Symbol: "$", Decimals: 2, Rate: 1},
	{Code: "EUR", Name: "Euro", Symbol: "€", Decimals: 2, Rate: 0.92},
	{Code: "GBP", Name: "British Pound", Symbol: "£", Decimals: 2, Rate: 0.79},
	{Code: "UAH", Name: "Ukrainian Hryvnia", Symbol: "₴", Decimals: 2, Rate: 41},
	{Code: "PLN", Name: "Polish Zloty", Symbol: "zł", Decimals: 2, Rate: 4},
	{Code: "JPY", Name: "Japanese Yen", Symbol: "¥", Decimals: 0, Rate: 150},
	{Code: "CHF", Name: "Swiss Franc", Symbol: "CHF", Decimals: 2, Rate: 0.88},
	{Code: "BTC", Name: "Bitcoin", Symbol: "₿", Decimals: 8, Rate: 0.000015},
}

func catalogItem(code string) *Currency {
	code = strings.ToUpper(code)
	for _, c := range builtinCatalog {
		if c.Code == code {
			cp := c
			return &cp
		}
	}
	return nil
}
