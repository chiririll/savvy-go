package seed

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"savvy-go/internal/domain"
	"savvy-go/internal/store"
)

var (
	usdCurrency = domain.Currency{Code: "USD", Name: "US Dollar", Symbol: "$", Decimals: 2, Rate: decimal.NewFromInt(1)}
	eurCurrency = domain.Currency{Code: "EUR", Name: "Euro", Symbol: "€", Decimals: 2, Rate: decimal.NewFromInt(1)}
)

// currencySet is the currencies of a space: base first, the others priced in it.
func currencySet(base domain.Currency, others ...domain.Currency) []domain.Currency {
	base.IsBase = true
	base.Rate = decimal.NewFromInt(1)
	return append([]domain.Currency{base}, others...)
}

func withRate(c domain.Currency, rate string) domain.Currency {
	c.Rate = decimal.RequireFromString(rate)
	return c
}

func seedReference(ctx context.Context, db store.DB, currencies []domain.Currency) error {
	if err := seedCurrencies(ctx, domain.Currencies{DB: db}, currencies); err != nil {
		return fmt.Errorf("currencies: %w", err)
	}
	if err := seedCategories(ctx, domain.Categories{DB: db}); err != nil {
		return fmt.Errorf("categories: %w", err)
	}
	if err := seedTags(ctx, domain.Tags{DB: db}); err != nil {
		return fmt.Errorf("tags: %w", err)
	}
	return nil
}

func seedCurrencies(ctx context.Context, curs domain.Currencies, list []domain.Currency) error {
	for _, c := range list {
		existing, err := curs.ByCode(ctx, c.Code)
		if err != nil {
			return err
		}
		if existing != nil {
			continue
		}
		if _, err := curs.Create(ctx, c); err != nil {
			return err
		}
	}
	return nil
}

func seedCategories(ctx context.Context, cats domain.Categories) error {
	for _, c := range defaultCategories {
		icon, color := c.icon, c.color
		isDefault := c.name == "Other" || c.name == "Other income"
		if _, err := cats.Create(ctx, domain.Category{
			Name: c.name, Type: c.typ, Icon: &icon, Color: &color, IsDefault: isDefault,
		}); err != nil {
			return err
		}
	}
	return nil
}

func seedTags(ctx context.Context, tags domain.Tags) error {
	for _, name := range defaultTags {
		if _, err := tags.Create(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

type seededCategory struct {
	name, typ, icon, color string
}

var defaultCategories = []seededCategory{
	{"Rent", "expense", "🏢", "#94a3b8"},
	{"Home", "expense", "🏠", "#a78bfa"},
	{"Utilities", "expense", "⚡", "#fbbf24"},
	{"Groceries", "expense", "🛒", "#4ade80"},
	{"Transport", "expense", "🚗", "#60a5fa"},
	{"Healthcare", "expense", "🏥", "#f87171"},
	{"Dining & takeout", "expense", "🍽️", "#fb923c"},
	{"Entertainment", "expense", "🎮", "#f472b6"},
	{"Shopping", "expense", "🛍️", "#2dd4bf"},
	{"Personal care", "expense", "✨", "#e879f9"},
	{"Gifts", "expense", "🎁", "#fb7185"},
	{"Travel", "expense", "✈️", "#38bdf8"},
	{"Other", "expense", "📌", "#94a3b8"},
	{"Salary", "income", "💵", "#4ade80"},
	{"Freelance", "income", "💻", "#60a5fa"},
	{"Investments", "income", "📈", "#a78bfa"},
	{"Gifts received", "income", "🎀", "#f472b6"},
	{"Refunds", "income", "↩️", "#2dd4bf"},
	{"Other income", "income", "💰", "#94a3b8"},
}

var defaultTags = []string{
	"Essential", "Optional", "Recurring", "One-time", "Business",
	"Personal", "Family", "Vacation", "Emergency", "Planned",
}
