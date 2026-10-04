package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/domain"
	"savvy-go/internal/store"
)

// miniSpace describes a small workspace: a few accounts, a handful of
// transactions placed relative to today, and a budget. The big Demo workspace
// is generated instead (see seedWorkspace).
type miniSpace struct {
	accounts []miniAccount
	txs      []miniTx
	budget   miniBudget
}

type miniAccount struct {
	key, name, typ, currency string
	balance                  float64
}

// miniTx is one transaction. daysAgo is negative for a date in the future; a
// pending transaction may carry an estimated amount.
type miniTx struct {
	typ, account, category string
	amount                 float64
	desc                   string
	daysAgo                int
	pending, estimated     bool
}

type miniBudget struct {
	name, category string
	amount         int64
}

var familySpace = miniSpace{
	accounts: []miniAccount{
		{"joint", "Joint Checking", "bank", "USD", 3200},
		{"kids", "Kids Savings", "bank", "USD", 1500},
		{"wallet", "Household Cash", "cash", "USD", 120},
	},
	txs: []miniTx{
		{"income", "joint", "#SALARY", 2800, "Jordan — Payroll", 33, false, false},
		{"income", "joint", "#SALARY", 2800, "Jordan — Payroll", 3, false, false},
		{"expense", "joint", "#GROCERIES", 164.30, "Whole Foods Market", 28, false, false},
		{"expense", "joint", "#GROCERIES", 128.75, "Trader Joe's", 14, false, false},
		{"expense", "wallet", "#DINING", 42, "Pizza night", 6, false, false},
		{"expense", "joint", "#HEALTH", 60, "Pediatrician copay", 18, false, false},
		{"expense", "joint", "#UTILITIES", 96.10, "ConEdison", 20, false, false},
		{"expense", "joint", "#UTILITIES", 105, "ConEdison — electricity bill (estimate)", -4, true, true},
		{"expense", "joint", "#HOUSING", 1450, "Summer camp deposit", -8, true, false},
	},
	budget: miniBudget{"Family Groceries", "#GROCERIES", 700},
}

var studioSpace = miniSpace{
	accounts: []miniAccount{
		{"bank", "Studio Account", "bank", "EUR", 12500},
		{"usd", "Studio USD Account", "bank", "USD", 3100},
	},
	txs: []miniTx{
		{"income", "bank", "#FREELANCE", 4800, "Client invoice — Nordlicht AG", 36, false, false},
		{"income", "usd", "#FREELANCE", 2200, "Client invoice — Brightpath Inc.", 21, false, false},
		{"income", "bank", "#FREELANCE", 3600, "Client invoice — Hafen Media", 5, false, false},
		{"expense", "bank", "#RENT", 950, "Office rent — Kreuzberg", 27, false, false},
		{"expense", "bank", "#RENT", 950, "Office rent — Kreuzberg", 1, false, false},
		{"expense", "bank", "#OTHER", 89, "Figma & Adobe subscriptions", 19, false, false},
		{"expense", "bank", "#UTILITIES", 74.50, "Vattenfall", 10, false, false},
		{"expense", "bank", "#OTHER", 2400, "VAT prepayment — Finanzamt", -6, true, true},
	},
	budget: miniBudget{"Office costs", "#RENT", 1000},
}

// seedMini fills a space with a miniSpace and returns its accounts by key. The
// space's reference data (currencies, categories) must already exist.
func seedMini(ctx context.Context, db store.DB, now time.Time, spec miniSpace) (map[string]*domain.Account, error) {
	curs := domain.Currencies{DB: db}
	accts := domain.Accounts{DB: db}
	txs := domain.Transactions{DB: db}
	cats := domain.Categories{DB: db}

	out := map[string]*domain.Account{}
	for _, a := range spec.accounts {
		cur, err := curs.ByCode(ctx, a.currency)
		if err != nil || cur == nil {
			return nil, fmt.Errorf("currency %s missing", a.currency)
		}
		created, err := accts.Create(ctx, domain.AccountInput{
			Name: a.name, Type: a.typ, CurrencyID: cur.ID,
			InitialBalance: decimal.NewFromFloat(a.balance), IsActive: true,
		})
		if err != nil {
			return nil, fmt.Errorf("account %s: %w", a.name, err)
		}
		out[a.key] = created
	}

	var all []domain.Category
	for _, typ := range []string{"expense", "income"} {
		list, err := cats.All(ctx, typ)
		if err != nil {
			return nil, err
		}
		all = append(all, list...)
	}
	for _, t := range spec.txs {
		in := domain.TxInput{
			Type: t.typ, AccountID: out[t.account].ID, Description: &t.desc,
			Amount: decimal.NewFromFloat(t.amount), IsEstimated: t.estimated,
		}
		if c := catByName(all, t.category); c != nil {
			in.CategoryID = &c.ID
		}
		date := now.AddDate(0, 0, -t.daysAgo).Format("2006-01-02")
		status := "confirmed"
		if t.pending {
			status = "pending"
		}
		in.Date, in.Status = &date, &status
		if _, err := txs.Create(ctx, in); err != nil {
			return nil, fmt.Errorf("transaction %s: %w", t.desc, err)
		}
	}

	if cat := catByName(all, spec.budget.category); cat != nil {
		cur, err := curs.Base(ctx)
		if err != nil {
			return nil, err
		}
		start := startOfMonth(now).Format("2006-01-02")
		notify := 80
		if _, err := (domain.Budgets{DB: db}).Create(ctx, domain.BudgetInput{
			Name: spec.budget.name, Amount: decimal.NewFromInt(spec.budget.amount), CurrencyID: &cur.ID,
			Period: "monthly", StartDate: &start, NotifyAtPercent: &notify, CategoryIDs: []int64{cat.ID},
		}); err != nil {
			return nil, err
		}
	}
	return out, nil
}
