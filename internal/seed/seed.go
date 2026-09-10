package seed

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"savvy-go/internal/auth"
	appdb "savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/domain"
	"savvy-go/internal/settings"
)

const demoSeededKey = "demo_seeded"

// Demo seeds currencies, categories, tags, and the full demo workspace when
// enabled and the database has never been demo-seeded (no users yet). Later
// starts are no-ops so first-boot matches Laravel's SEED_DEMO behavior.
func Demo(ctx context.Context, db *sql.DB, enabled bool, loc *time.Location) error {
	if !enabled {
		return nil
	}
	if loc == nil {
		loc = time.UTC
	}
	st := settings.Store{DB: db}
	if st.Bool(ctx, demoSeededKey, false) {
		return nil
	}
	n, err := (auth.Users{DB: db}).Count(ctx)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if n > 0 {
		return nil
	}

	if err := seedReference(ctx, db); err != nil {
		return err
	}
	if err := seedWorkspace(ctx, db, loc); err != nil {
		return err
	}
	if err := st.Set(ctx, demoSeededKey, true); err != nil {
		return fmt.Errorf("mark demo seeded: %w", err)
	}

	txs, _ := appdb.Q(db).CountTransactions(ctx, sqlc.CountTransactionsParams{})
	accts, _ := appdb.Q(db).CountAccounts(ctx)
	slog.Info("demo data seeded", "transactions", txs, "accounts", accts)
	return nil
}

func seedReference(ctx context.Context, db *sql.DB) error {
	if err := seedCurrencies(ctx, domain.Currencies{DB: db}); err != nil {
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

func seedCurrencies(ctx context.Context, curs domain.Currencies) error {
	for _, c := range []domain.Currency{
		{Code: "USD", Name: "US Dollar", Symbol: "$", Decimals: 2, IsBase: true, Rate: 1},
		{Code: "EUR", Name: "Euro", Symbol: "€", Decimals: 2, Rate: 1.08},
	} {
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
		if _, err := cats.Create(ctx, domain.Category{
			Name: c.name, Type: c.typ, Icon: &icon, Color: &color,
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
