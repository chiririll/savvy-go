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
		isDefault := c.name == "#OTHER" || c.name == "#OTHER_INCOME"
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
	{"#RENT", "expense", "🏢", "#94a3b8"},
	{"#HOUSING", "expense", "🏠", "#a78bfa"},
	{"#UTILITIES", "expense", "⚡", "#fbbf24"},
	{"#GROCERIES", "expense", "🛒", "#4ade80"},
	{"#TRANSPORT", "expense", "🚗", "#60a5fa"},
	{"#HEALTH", "expense", "🏥", "#f87171"},
	{"#DINING", "expense", "🍽️", "#fb923c"},
	{"#ENTERTAINMENT", "expense", "🎮", "#f472b6"},
	{"#SHOPPING", "expense", "🛍️", "#2dd4bf"},
	{"#PERSONAL_CARE", "expense", "✨", "#e879f9"},
	{"#GIFTS", "expense", "🎁", "#fb7185"},
	{"#TRAVEL", "expense", "✈️", "#38bdf8"},
	{"#OTHER", "expense", "📌", "#94a3b8"},
	{"#SALARY", "income", "💵", "#4ade80"},
	{"#FREELANCE", "income", "💻", "#60a5fa"},
	{"#INVESTMENTS", "income", "📈", "#a78bfa"},
	{"#GIFTS_RECEIVED", "income", "🎀", "#f472b6"},
	{"#REFUNDS", "income", "↩️", "#2dd4bf"},
	{"#OTHER_INCOME", "income", "💰", "#94a3b8"},
}

var defaultTags = []string{
	"Essential", "Optional", "Recurring", "One-time", "Business",
	"Personal", "Family", "Vacation", "Emergency", "Planned",
}
