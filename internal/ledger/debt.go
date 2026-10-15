package ledger

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/money"
)

type Debts struct {
	Accounts     Accounts
	Transactions Transactions
}

func (s Debts) repo() repo.Repo { return repo.Repo{DB: s.Accounts.DB} }

func (s Debts) All(ctx context.Context, includeCompleted bool) ([]Account, error) {
	return s.Accounts.Debts(ctx, includeCompleted)
}

// DebtInput describes a new debt. Origin "new" with an AccountID records the
// issuance as a transaction on that account and takes its currency.
type DebtInput struct {
	Name         string
	DebtType     string
	CurrencyID   int64
	AccountID    int64
	Amount       decimal.Decimal
	Date         string
	Origin       string
	DueDate      *string
	Counterparty *string
	Description  *string
}

func (s Debts) Create(ctx context.Context, in DebtInput) (*Account, error) {
	name, debtType, currencyID, accountID := in.Name, in.DebtType, in.CurrencyID, in.AccountID
	amount, date, origin := in.Amount, in.Date, in.Origin
	due, counter, desc := in.DueDate, in.Counterparty, in.Description
	if origin == "new" && accountID != 0 {
		src, err := s.Accounts.ByID(ctx, accountID)
		if err != nil || src == nil {
			return nil, fmt.Errorf("source account")
		}
		if src.Type == "debt" {
			return nil, fmt.Errorf("cannot use debt source")
		}
		currencyID = src.CurrencyID
	}
	debt, err := s.Accounts.Create(ctx, AccountInput{
		Name: name, Type: "debt", CurrencyID: currencyID,
		InitialBalance: decimal.Zero, TargetAmount: &amount, DueDate: due,
		Counterparty: counter, DebtDesc: desc, IsActive: true, DebtType: &debtType,
	})
	if err != nil {
		return nil, err
	}
	if origin == "new" && accountID != 0 {
		if err := s.recordIssuance(ctx, *debt, accountID, debtType, amount, date, desc); err != nil {
			return nil, err
		}
		debt, err = s.Accounts.ByID(ctx, debt.ID)
		if err != nil {
			return nil, err
		}
	}
	return debt, nil
}

func (s Debts) recordIssuance(ctx context.Context, debt Account, accountID int64, debtType string, amount decimal.Decimal, date string, desc *string) error {
	typ := "debt_lend"
	amt := amount
	toAmt := amount
	if debtType == "i_owe" {
		typ = "debt_borrow"
	}
	_, err := s.Transactions.Create(ctx, TxInput{
		Type: typ, AccountID: accountID, ToAccountID: &debt.ID,
		Amount: amt, ToAmount: &toAmt, Description: desc, Date: &date,
	})
	return err
}

func (s Debts) Payment(ctx context.Context, debtID, accountID int64, amount decimal.Decimal, date string, desc *string, collect bool) (*Transaction, error) {
	debt, err := s.Accounts.ByID(ctx, debtID)
	if err != nil || debt == nil || debt.Type != "debt" {
		return nil, fmt.Errorf("not a debt")
	}
	if debt.IsPaidOff {
		return nil, fmt.Errorf("already paid off")
	}
	typ := "debt_payment"
	if collect {
		typ = "debt_collection"
	}
	tx, err := s.Transactions.Create(ctx, TxInput{
		Type: typ, AccountID: accountID, ToAccountID: &debt.ID,
		Amount: amount, ToAmount: &amount, Description: desc, Date: &date,
	})
	if err != nil {
		return nil, err
	}
	s.maybePayOff(ctx, debt)
	return tx, nil
}

func (s Debts) maybePayOff(ctx context.Context, debt *Account) {
	fresh, err := s.Accounts.ByID(ctx, debt.ID)
	if err != nil || fresh == nil {
		return
	}
	if !fresh.Balance.IsPositive() {
		_ = s.repo().MarkPaidOff(ctx, debt.ID)
	}
}

func (s Debts) Reopen(ctx context.Context, id int64) (*Account, error) {
	d, err := s.Accounts.ByID(ctx, id)
	if err != nil || d == nil || d.Type != "debt" {
		return nil, fmt.Errorf("not a debt")
	}
	if !d.IsPaidOff {
		return nil, fmt.Errorf("not paid off")
	}
	if err := s.repo().Reopen(ctx, id, time.Now().UTC()); err != nil {
		return nil, err
	}
	return s.Accounts.ByID(ctx, id)
}

// DebtSummary totals open debts in the base currency (nil when there is none),
// rounded to its decimals.
type DebtSummary struct {
	IOwe     money.Money
	OwedToMe money.Money
	Count    int
	Currency *Currency
}

func (s Debts) Summary(ctx context.Context) DebtSummary {
	debts, _ := s.All(ctx, false)
	base, _ := (Currencies{DB: s.Accounts.DB}).Base(ctx)
	out := DebtSummary{Count: len(debts), Currency: base}
	if base == nil {
		return out
	}
	out.IOwe, out.OwedToMe = money.Zero(base.Unit()), money.Zero(base.Unit())
	for _, d := range debts {
		amt := Convert(d.Balance, *d.Currency, *base)
		if d.DebtType != nil && *d.DebtType == "i_owe" {
			out.IOwe = out.IOwe.Add(amt)
		} else {
			out.OwedToMe = out.OwedToMe.Add(amt)
		}
	}
	return out
}

func (s Debts) Delete(ctx context.Context, id int64) error {
	if s.repo().CountDebtHistory(ctx, id) > 0 {
		return fmt.Errorf("has history")
	}
	return s.repo().DeleteDebt(ctx, id)
}
