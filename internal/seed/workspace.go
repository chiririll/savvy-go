package seed

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/automation"
	"savvy-go/internal/ledger"
	"savvy-go/internal/recurring"
	"savvy-go/internal/store"
)

type seeder struct {
	ctx      context.Context
	db       store.DB
	now      time.Time
	start    time.Time
	end      time.Time
	rng      *rand.Rand
	txs      ledger.Transactions
	accts    ledger.Accounts
	cats     ledger.Categories
	tags     ledger.Tags
	budgets  ledger.Budgets
	recur    recurring.Service
	auto     automation.Service
	expenses []expenseCand
}

type expenseCand struct {
	id      int64
	amount  float64
	catName string
}

func seedWorkspace(ctx context.Context, db store.DB, now time.Time) (map[string]*ledger.Account, error) {
	start := startOfMonth(now.AddDate(0, -monthsOfHistory, 0))
	s := &seeder{
		ctx:     ctx,
		db:      db,
		now:     now,
		start:   start,
		end:     now,
		rng:     rand.New(rand.NewSource(rngSeed)),
		txs:     ledger.Transactions{DB: db},
		accts:   ledger.Accounts{DB: db},
		cats:    ledger.Categories{DB: db},
		tags:    ledger.Tags{DB: db},
		budgets: ledger.Budgets{DB: db},
		recur:   recurring.Service{DB: db, Txs: ledger.Transactions{DB: db}},
		auto:    automation.Service{DB: db, Txs: ledger.Transactions{DB: db}},
	}
	curs := ledger.Currencies{DB: db}
	usd, err := curs.ByCode(ctx, "USD")
	if err != nil || usd == nil {
		return nil, fmt.Errorf("USD currency missing")
	}
	eur, err := curs.ByCode(ctx, "EUR")
	if err != nil {
		return nil, err
	}
	accounts, err := s.createAccounts(usd, eur)
	if err != nil {
		return nil, err
	}
	expenses, err := s.cats.All(ctx, "expense")
	if err != nil {
		return nil, err
	}
	incomes, err := s.cats.All(ctx, "income")
	if err != nil {
		return nil, err
	}
	tags, err := s.tags.All(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.createTransactions(accounts, expenses, incomes, tags); err != nil {
		return nil, err
	}
	if err := s.seedTransactionItems(); err != nil {
		return nil, err
	}
	if err := s.seedPending(accounts, expenses); err != nil {
		return nil, err
	}
	if err := s.createBudgets(usd, expenses, tags); err != nil {
		return nil, err
	}
	if err := s.createRecurring(accounts, expenses, incomes); err != nil {
		return nil, err
	}
	return accounts, s.createAutomation(tags)
}

// accountSpec describes one demo account; the debt fields are empty for the others.
type accountSpec struct {
	key, name, typ string
	cur            *ledger.Currency
	bal            float64
	debtType       string
	target         *float64
	due            *string
	counter, desc  string
}

func (s *seeder) accountSpecs(usd, eur *ledger.Currency) []accountSpec {
	mortgageDue := s.now.AddDate(25, 0, 0).Format("2006-01-02")
	loanDue := s.now.AddDate(0, 4, 0).Format("2006-01-02")
	zero, mortgageAmt, loanAmt := 0.0, 285000.00, 1200.00

	specs := []accountSpec{
		{key: "checking", name: "Chase Checking", typ: "bank", cur: usd, bal: 8200},
		{key: "cash", name: "Cash Wallet", typ: "cash", cur: usd, bal: 340},
		{key: "savings", name: "Ally Savings", typ: "bank", cur: usd, bal: 24500},
		{key: "crypto", name: "Coinbase Portfolio", typ: "crypto", cur: usd, bal: 6300},
	}
	if eur != nil {
		specs = append(specs, accountSpec{key: "eur", name: "Revolut EUR", typ: "bank", cur: eur, bal: 1850})
	}
	return append(specs,
		accountSpec{
			key: "credit", name: "Amex Gold Card", typ: "debt", cur: usd, debtType: "i_owe", target: &zero,
			counter: "American Express", desc: "Credit card balance",
		},
		accountSpec{
			key: "mortgage", name: "Home Mortgage", typ: "debt", cur: usd, debtType: "i_owe", target: &mortgageAmt,
			due: &mortgageDue, counter: "Wells Fargo Home Mortgage", desc: "30-year fixed mortgage",
		},
		accountSpec{
			key: "loan_out", name: "Loan to Michael", typ: "debt", cur: usd, debtType: "owed_to_me", target: &loanAmt,
			due: &loanDue, counter: "Michael Chen", desc: "Helped with moving costs",
		},
	)
}

func (s *seeder) createAccounts(usd *ledger.Currency, eur *ledger.Currency) (map[string]*ledger.Account, error) {
	out := map[string]*ledger.Account{}
	for _, sp := range s.accountSpecs(usd, eur) {
		a := ledger.AccountInput{
			Name: sp.name, Type: sp.typ, CurrencyID: sp.cur.ID,
			InitialBalance: decimal.NewFromFloat(sp.bal), IsActive: true,
		}
		if sp.debtType != "" {
			a.DebtType = &sp.debtType
			if sp.target != nil {
				target := decimal.NewFromFloat(*sp.target)
				a.TargetAmount = &target
			}
			a.DueDate = sp.due
			a.Counterparty = strPtr(sp.counter)
			a.DebtDesc = strPtr(sp.desc)
		}
		created, err := s.accts.Create(s.ctx, a)
		if err != nil {
			return nil, fmt.Errorf("account %s: %w", sp.name, err)
		}
		out[sp.key] = created
	}
	return out, nil
}

func (s *seeder) createTransactions(
	accounts map[string]*ledger.Account,
	expenses, incomes []ledger.Category,
	tags []ledger.Tag,
) error {
	if err := s.seedIncome(accounts, incomes); err != nil {
		return err
	}
	if err := s.seedFixedExpenses(accounts, expenses); err != nil {
		return err
	}
	if err := s.seedVariableExpenses(accounts, expenses, tags); err != nil {
		return err
	}
	if err := s.seedSplitPayments(accounts, expenses); err != nil {
		return err
	}
	return s.seedTransfersDebt(accounts)
}
