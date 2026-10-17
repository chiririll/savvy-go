package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
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
func seedMini(ctx context.Context, db store.DB, now time.Time, spec miniSpace) (map[string]*ledger.Account, error) {
	out, err := seedMiniAccounts(ctx, db, spec.accounts)
	if err != nil {
		return nil, err
	}
	cats := ledger.Categories{DB: db}
	var all []ledger.Category
	for _, typ := range []string{"expense", "income"} {
		list, err := cats.All(ctx, typ)
		if err != nil {
			return nil, err
		}
		all = append(all, list...)
	}
	txs := ledger.Transactions{DB: db}
	for _, t := range spec.txs {
		if _, err := txs.Create(ctx, miniTxInput(t, out[t.account].ID, all, now)); err != nil {
			return nil, fmt.Errorf("transaction %s: %w", t.desc, err)
		}
	}
	if cat := catByName(all, spec.budget.category); cat != nil {
		if err := seedMiniBudget(ctx, db, now, spec.budget, cat.ID); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func seedMiniAccounts(ctx context.Context, db store.DB, specs []miniAccount) (map[string]*ledger.Account, error) {
	curs := ledger.Currencies{DB: db}
	accts := ledger.Accounts{DB: db}
	out := map[string]*ledger.Account{}
	for _, a := range specs {
		cur, err := curs.ByCode(ctx, a.currency)
		if err != nil || cur == nil {
			return nil, fmt.Errorf("currency %s missing", a.currency)
		}
		created, err := accts.Create(ctx, ledger.AccountInput{
			Name: a.name, Type: a.typ, CurrencyID: cur.ID,
			InitialBalance: decimal.NewFromFloat(a.balance), IsActive: true,
		})
		if err != nil {
			return nil, fmt.Errorf("account %s: %w", a.name, err)
		}
		out[a.key] = created
	}
	return out, nil
}

func miniTxInput(t miniTx, accountID int64, cats []ledger.Category, now time.Time) ledger.TxInput {
	in := ledger.TxInput{
		Type: t.typ, AccountID: accountID, Description: &t.desc,
		Amount: decimal.NewFromFloat(t.amount), IsEstimated: t.estimated,
	}
	if c := catByName(cats, t.category); c != nil {
		in.CategoryID = &c.ID
	}
	date := now.AddDate(0, 0, -t.daysAgo).Format("2006-01-02")
	status := "confirmed"
	if t.pending {
		status = "pending"
	}
	in.Date, in.Status = &date, &status
	return in
}

func seedMiniBudget(ctx context.Context, db store.DB, now time.Time, b miniBudget, categoryID int64) error {
	cur, err := (ledger.Currencies{DB: db}).Base(ctx)
	if err != nil {
		return err
	}
	start := startOfMonth(now).Format("2006-01-02")
	notify := 80
	_, err = (ledger.Budgets{DB: db}).Create(ctx, ledger.BudgetInput{
		Name: b.name, Amount: decimal.NewFromInt(b.amount), CurrencyID: &cur.ID,
		Period: "monthly", StartDate: &start, NotifyAtPercent: &notify, CategoryIDs: []int64{categoryID},
	})
	return err
}
