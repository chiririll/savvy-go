package dto

import (
	"time"

	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
)

type Account struct {
	ID             int64       `json:"id"`
	Name           string      `json:"name"`
	Type           string      `json:"type"`
	InitialBalance money.Money `json:"initialBalance"`
	CurrentBalance money.Money `json:"currentBalance"`
	IsActive       bool        `json:"isActive"`
	SortOrder      int         `json:"sortOrder"`
	Currency       *Currency   `json:"currency"`
	CreatedAt      *time.Time  `json:"createdAt"`
}

func NewAccount(a ledger.Account) Account {
	return Account{
		ID: a.ID, Name: a.Name, Type: a.Type,
		InitialBalance: a.InitialBalance, CurrentBalance: a.Balance,
		IsActive: a.IsActive, SortOrder: a.SortOrder,
		Currency: ptr(a.Currency, NewCurrency), CreatedAt: utc(a.CreatedAt),
	}
}

// Debt is a debt account. CurrentBalance is the amount still owed.
type Debt struct {
	Account
	DebtType        *string     `json:"debtType"`
	TargetAmount    money.Money `json:"targetAmount"`
	PaymentProgress float64     `json:"paymentProgress"`
	DueDate         *string     `json:"dueDate"`
	Counterparty    *string     `json:"counterparty"`
	Description     *string     `json:"description"`
	IsPaidOff       bool        `json:"isPaidOff"`
}

func NewDebt(a ledger.Account) Debt {
	return Debt{
		Account: NewAccount(a), DebtType: a.DebtType, TargetAmount: a.DebtTarget(), PaymentProgress: a.PaymentProgress(),
		DueDate: a.DueDate, Counterparty: a.Counterparty, Description: a.DebtDesc, IsPaidOff: a.IsPaidOff,
	}
}

type AccountsSummary struct {
	TotalBalance  money.Money `json:"totalBalance"`
	Currency      *string     `json:"currency"`
	Decimals      int         `json:"decimals"`
	AccountsCount int         `json:"accountsCount"`
}

func NewAccountsSummary(s ledger.AccountsSummary) AccountsSummary {
	return AccountsSummary{
		TotalBalance: s.Total, Currency: currencyCode(s.Currency), Decimals: currencyDecimals(s.Currency), AccountsCount: s.Count,
	}
}

type DebtSummary struct {
	TotalIOwe     money.Money `json:"totalIOwe"`
	TotalOwedToMe money.Money `json:"totalOwedToMe"`
	NetDebt       money.Money `json:"netDebt"`
	DebtsCount    int         `json:"debtsCount"`
	Currency      *string     `json:"currency"`
	Decimals      int         `json:"decimals"`
}

func NewDebtSummary(s ledger.DebtSummary) DebtSummary {
	return DebtSummary{
		TotalIOwe: s.IOwe, TotalOwedToMe: s.OwedToMe, NetDebt: s.Net(), DebtsCount: s.Count,
		Currency: currencyCode(s.Currency), Decimals: currencyDecimals(s.Currency),
	}
}

func currencyCode(c *ledger.Currency) *string {
	if c == nil {
		return nil
	}
	return &c.Code
}

// currencyDecimals is the scale totals in c are rounded to (2 without one).
func currencyDecimals(c *ledger.Currency) int {
	if c == nil {
		return 2
	}
	return c.Decimals
}
