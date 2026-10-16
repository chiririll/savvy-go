package dto

import (
	"time"

	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
)

type Currency struct {
	ID       int64  `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
	IsBase   bool   `json:"isBase"`
	Rate     Number `json:"rate"`
}

func NewCurrency(c ledger.Currency) Currency {
	return Currency{ID: c.ID, Code: c.Code, Name: c.Name, Symbol: c.Symbol, Decimals: c.Decimals, IsBase: c.IsBase, Rate: Number(c.Rate)}
}

type Tag struct {
	ID                int64      `json:"id"`
	Name              string     `json:"name"`
	TransactionsCount int        `json:"transactionsCount"`
	CreatedAt         *time.Time `json:"createdAt"`
}

func NewTag(t ledger.Tag) Tag {
	return Tag{ID: t.ID, Name: t.Name, TransactionsCount: t.TransactionsCount, CreatedAt: utc(t.CreatedAt)}
}

type Category struct {
	ID                int64        `json:"id"`
	Name              string       `json:"name"`
	Type              string       `json:"type"`
	Icon              *string      `json:"icon"`
	Color             *string      `json:"color"`
	IsDefault         bool         `json:"isDefault"`
	TransactionsCount int          `json:"transactionsCount"`
	TotalAmount       *money.Money `json:"totalAmount"`
}

func NewCategory(c ledger.Category) Category {
	return Category{
		ID: c.ID, Name: c.Name, Type: c.Type, Icon: c.Icon, Color: c.Color, IsDefault: c.IsDefault,
		TransactionsCount: c.TransactionsCount, TotalAmount: c.TotalAmount,
	}
}

type CategoryStatistics struct {
	CategoryID        int64       `json:"categoryId"`
	CategoryName      string      `json:"categoryName"`
	Type              string      `json:"type"`
	TransactionsCount int         `json:"transactionsCount"`
	TotalAmount       money.Money `json:"totalAmount"`
}

func NewCategoryStatistics(s ledger.CategoryStatistics) CategoryStatistics {
	return CategoryStatistics{
		CategoryID: s.Category.ID, CategoryName: s.Category.Name, Type: s.Category.Type,
		TransactionsCount: s.Count, TotalAmount: s.Total,
	}
}

// CatalogCurrency is a currency that can be added: the same shape for a
// space's catalog and the server-wide one.
type CatalogCurrency struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
	Rate     Number `json:"rate"`
}

func NewCatalogCurrency(c ledger.Currency) CatalogCurrency {
	return CatalogCurrency{Code: c.Code, Name: c.Name, Symbol: c.Symbol, Decimals: c.Decimals, Rate: Number(c.Rate)}
}

// CategorySummary lists categories with what they hold in a period, in the
// base currency.
type CategorySummary struct {
	Data     []Category  `json:"data"`
	Total    money.Money `json:"total"`
	Currency string      `json:"currency"`
}

// CurrencyConversion is an amount converted between two of the space's currencies.
type CurrencyConversion struct {
	Amount money.Money `json:"amount"`
	From   Currency    `json:"from"`
	To     Currency    `json:"to"`
	Result money.Money `json:"result"`
}
