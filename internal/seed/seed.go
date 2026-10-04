package seed

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/auth"
	appdb "savvy-go/internal/db"
	"savvy-go/internal/domain"
	"savvy-go/internal/settings"
	"savvy-go/internal/store"
)

const demoSeededKey = "demo_seeded"

// Demo seeds the demo users and a "Demo" space with currencies, categories,
// tags and a full workspace when enabled and the server has never been
// demo-seeded (no users yet). Later starts are no-ops so first-boot matches
// Laravel's SEED_DEMO behavior. The admin owns the space, the editor edits it
// and the demo user only views it.
func Demo(ctx context.Context, st store.Store, enabled bool, loc *time.Location) error {
	if !enabled {
		return nil
	}
	if loc == nil {
		loc = time.UTC
	}
	server := settings.Store{DB: st.Server()}
	if server.Bool(ctx, demoSeededKey, false) {
		return nil
	}
	users := auth.Users{DB: st.Server()}
	n, err := users.Count(ctx)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if n > 0 {
		return nil
	}

	members := map[string]*auth.User{}
	for _, u := range []struct {
		name, email, pass, role, spaceRole string
	}{
		{"Alex Morgan", "admin@savvy.app", "password", auth.RoleAdmin, domain.SpaceAdmin},
		{"Jordan Lee", "editor@savvy.app", "password", auth.RoleUser, domain.SpaceEditor},
		{"Demo User", "demo@demo.com", "demo", auth.RoleUser, domain.SpaceViewer},
	} {
		pass := u.pass
		created, err := users.Create(ctx, u.name, u.email, &pass, u.role)
		if err != nil {
			return fmt.Errorf("user %s: %w", u.email, err)
		}
		members[u.spaceRole] = created
	}
	spaces := domain.Spaces{Store: st}
	space, err := spaces.Create(ctx, "Demo", members[domain.SpaceAdmin])
	if err != nil {
		return fmt.Errorf("demo space: %w", err)
	}
	for _, role := range []string{domain.SpaceEditor, domain.SpaceViewer} {
		if err := spaces.SetMember(ctx, space.ID, members[role], role); err != nil {
			return err
		}
	}
	db, err := st.Space(ctx, space.ID)
	if err != nil {
		return err
	}

	if err := seedReference(ctx, db); err != nil {
		return err
	}
	if err := seedWorkspace(ctx, db, loc); err != nil {
		return err
	}
	if err := server.Set(ctx, demoSeededKey, true); err != nil {
		return fmt.Errorf("mark demo seeded: %w", err)
	}

	txs, _ := appdb.Q(db).CountAllTransactions(ctx)
	accts, _ := appdb.Q(db).CountAccounts(ctx)
	slog.Info("demo data seeded", "transactions", txs, "accounts", accts)
	return nil
}

func seedReference(ctx context.Context, db store.DB) error {
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
		{Code: "USD", Name: "US Dollar", Symbol: "$", Decimals: 2, IsBase: true, Rate: decimal.NewFromInt(1)},
		{Code: "EUR", Name: "Euro", Symbol: "€", Decimals: 2, Rate: decimal.RequireFromString("1.08")},
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
