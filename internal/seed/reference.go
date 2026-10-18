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

func seedReference(ctx context.Context, db store.DB, currencies []ledger.Currency) error {
	if err := seedCurrencies(ctx, ledger.Currencies{DB: db}, currencies); err != nil {
		return fmt.Errorf("currencies: %w", err)
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

func seedCategories(ctx context.Context, cats ledger.Categories) error {
	for _, c := range DefaultCategories() {
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
	for _, name := range DefaultTags() {
		if _, err := tags.Create(ctx, name); err != nil {
			return err
		}
	}
	return nil
}
