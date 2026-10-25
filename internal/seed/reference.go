package seed

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
	"savvy-go/internal/store"
)

var (
	usdCurrency = ledger.Currency{Code: "USD", Name: "US Dollar", Symbol: "$", Decimals: 2, Rate: decimal.NewFromInt(1)}
	eurCurrency = ledger.Currency{Code: "EUR", Name: "Euro", Symbol: "€", Decimals: 2, Rate: decimal.NewFromInt(1)}
)

// currencySet is the currencies of a space: base first, the others priced in it.
func currencySet(base ledger.Currency, others ...ledger.Currency) []ledger.Currency {
	base.IsBase = true
	base.Rate = decimal.NewFromInt(1)
	return append([]ledger.Currency{base}, others...)
}

func withRate(c ledger.Currency, rate string) ledger.Currency {
	c.Rate = decimal.RequireFromString(rate)
	return c
}

// seedReference seeds a space's currencies and, unless the space brings its own
// (ownReference.replacesPresets), the preset categories and tags.
func seedReference(ctx context.Context, db store.DB, currencies []ledger.Currency, presets bool) error {
	if err := seedCurrencies(ctx, ledger.Currencies{DB: db}, currencies); err != nil {
		return fmt.Errorf("currencies: %w", err)
	}
	if !presets {
		return nil
	}
	if err := seedCategories(ctx, ledger.Categories{DB: db}); err != nil {
		return fmt.Errorf("categories: %w", err)
	}
	if err := seedTags(ctx, ledger.Tags{DB: db}); err != nil {
		return fmt.Errorf("tags: %w", err)
	}
	return nil
}

func seedCurrencies(ctx context.Context, curs ledger.Currencies, list []ledger.Currency) error {
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

// ownReference is what a space has besides the presets, or instead of them when
// replacesPresets is set.
type ownReference struct {
	categories      []Category
	tags            []string
	replacesPresets bool
}

func seedOwn(ctx context.Context, db store.DB, own ownReference) error {
	if err := createCategories(ctx, ledger.Categories{DB: db}, own.categories); err != nil {
		return fmt.Errorf("categories: %w", err)
	}
	if err := createTags(ctx, ledger.Tags{DB: db}, own.tags); err != nil {
		return fmt.Errorf("tags: %w", err)
	}
	return nil
}

func seedCategories(ctx context.Context, cats ledger.Categories) error {
	return createCategories(ctx, cats, DefaultCategories())
}

func createCategories(ctx context.Context, cats ledger.Categories, list []Category) error {
	for _, c := range list {
		icon, color := c.Icon, c.Color
		if _, err := cats.Create(ctx, ledger.Category{
			Name: c.Name, Type: c.Type, Icon: &icon, Color: &color, IsDefault: c.IsDefault,
		}); err != nil {
			return err
		}
	}
	return nil
}

func seedTags(ctx context.Context, tags ledger.Tags) error {
	return createTags(ctx, tags, DefaultTags())
}

func createTags(ctx context.Context, tags ledger.Tags, names []string) error {
	for _, name := range names {
		if _, err := tags.Create(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

// studioReference is the categories and tags of Studio GmbH, a business with
// nothing in common with the household presets.
var studioReference = ownReference{
	replacesPresets: true,
	categories: []Category{
		{"Rent & office", "expense", "🏢", "#a78bfa", false},
		{"Software & tools", "expense", "🧰", "#818cf8", false},
		{"Contractors", "expense", "🤝", "#60a5fa", false},
		{"Marketing", "expense", "📣", "#f472b6", false},
		{"Hardware", "expense", "🖥️", "#2dd4bf", false},
		{"Taxes & VAT", "expense", "🧾", "#fbbf24", false},
		{"Events & travel", "expense", "🚆", "#38bdf8", false},
		{"Other costs", "expense", "📌", "#94a3b8", true},
		{"Client projects", "income", "🎨", "#4ade80", false},
		{"Retainers", "income", "🔁", "#34d399", false},
		{"Licensing", "income", "📜", "#a3e635", false},
		{"Other revenue", "income", "💰", "#94a3b8", true},
	},
	tags: []string{"Client work", "Billable", "Overhead", "Reimbursable", "Q4 deadline"},
}
