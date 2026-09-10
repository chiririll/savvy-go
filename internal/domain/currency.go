package domain

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/chiririll/savvy-plus/internal/db"
	"github.com/chiririll/savvy-plus/internal/db/sqlc"
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
	rows, err := db.Q(s.DB).ListCurrencies(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Currency, 0, len(rows))
	for _, r := range rows {
		out = append(out, currencyFrom(r.ID, r.Code, r.Name, r.Symbol, r.Decimals, r.IsBase, r.Rate))
	}
	return out, nil
}

func (s Currencies) ByID(ctx context.Context, id int64) (*Currency, error) {
	r, err := db.Q(s.DB).GetCurrency(ctx, id)
	return currencyFromRow(r.ID, r.Code, r.Name, r.Symbol, r.Decimals, r.IsBase, r.Rate, err)
}

func (s Currencies) ByCode(ctx context.Context, code string) (*Currency, error) {
	r, err := db.Q(s.DB).GetCurrencyByCode(ctx, strings.ToUpper(code))
	return currencyFromRow(r.ID, r.Code, r.Name, r.Symbol, r.Decimals, r.IsBase, r.Rate, err)
}

func (s Currencies) Base(ctx context.Context) (*Currency, error) {
	r, err := db.Q(s.DB).GetBaseCurrency(ctx)
	return currencyFromRow(r.ID, r.Code, r.Name, r.Symbol, r.Decimals, r.IsBase, r.Rate, err)
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
		if err := db.Q(s.DB).ClearBaseCurrency(ctx); err != nil {
			return nil, err
		}
		c.Rate = 1
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Q(s.DB).InsertCurrency(ctx, sqlc.InsertCurrencyParams{
		Code: c.Code, Name: c.Name, Symbol: c.Symbol, Decimals: int64(c.Decimals),
		IsBase: db.BoolInt(c.IsBase), Rate: c.Rate, CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
	})
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
		if err := db.Q(s.DB).ClearBaseCurrency(ctx); err != nil {
			return nil, err
		}
		c.Rate = 1
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err = db.Q(s.DB).UpdateCurrency(ctx, sqlc.UpdateCurrencyParams{
		Name: c.Name, Symbol: c.Symbol, Decimals: int64(c.Decimals), IsBase: db.BoolInt(c.IsBase),
		Rate: c.Rate, UpdatedAt: db.NS(now), ID: id,
	})
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
	used, _ := db.Q(s.DB).CountAccountsForCurrency(ctx, id)
	if used > 0 {
		return fmt.Errorf("in use")
	}
	if cur.IsBase {
		return fmt.Errorf("base")
	}
	n, _ := db.Q(s.DB).CountCurrencies(ctx)
	if n <= 1 {
		return fmt.Errorf("last")
	}
	return db.Q(s.DB).DeleteCurrency(ctx, id)
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
	rows, err := db.Q(s.DB).ListOtherCurrencyRates(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if err := db.Q(s.DB).UpdateCurrencyRate(ctx, sqlc.UpdateCurrencyRateParams{Rate: r.Rate / newRate, ID: r.ID}); err != nil {
			return nil, err
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err = db.Q(s.DB).SetCurrencyBase(ctx, sqlc.SetCurrencyBaseParams{UpdatedAt: db.NS(now), ID: id})
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
	n, _ := db.Q(s.DB).CountCurrencies(ctx)
	item.IsBase = n == 0
	if item.IsBase {
		item.Rate = 1
	}
	return s.Create(ctx, *item)
}

func (s Currencies) Catalog(ctx context.Context) []map[string]any {
	existing := map[string]bool{}
	if rows, err := db.Q(s.DB).ListCurrencyCodes(ctx); err == nil {
		for _, code := range rows {
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

func currencyFrom(id int64, code, name, symbol string, decimals, isBase int64, rate float64) Currency {
	return Currency{ID: id, Code: code, Name: name, Symbol: symbol, Decimals: int(decimals), IsBase: isBase != 0, Rate: rate}
}

func currencyFromRow(id int64, code, name, symbol string, decimals, isBase int64, rate float64, err error) (*Currency, error) {
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c := currencyFrom(id, code, name, symbol, decimals, isBase, rate)
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
