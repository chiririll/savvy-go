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

type AccountsSummary struct {
	TotalBalance  money.Money `json:"totalBalance"`
	Currency      *string     `json:"currency"`
	Decimals      int         `json:"decimals"`
	AccountsCount int         `json:"accountsCount"`
}

func NewAccountsSummary(s ledger.AccountsSummary) AccountsSummary {
	return AccountsSummary{
		TotalBalance: s.Total, Currency: currencyCode(s.Currency), Decimals: currencyDecimals(s.Currency),
		AccountsCount: s.Count,
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

// AccountList is accounts with the summary the caller asked for.
type AccountList struct {
	Data    []Account       `json:"data"`
	Summary AccountsSummary `json:"summary"`
}

// BalanceSeries is the balance of one account, or of all, over the dates of a
// BalanceHistory. Data is in the base currency; NativeData is in the account's
// own and absent for the total.
type BalanceSeries struct {
	ID         *int64        `json:"id"`
	Name       string        `json:"name"`
	Type       string        `json:"type"`
	Data       []money.Money `json:"data"`
	NativeData []money.Money `json:"native_data,omitempty"`
	Currency   string        `json:"currency"`
}

// BalanceHistory has snake_case keys inside its series, like the chart client
// that reads them.
type BalanceHistory struct {
	Dates    []string        `json:"dates"`
	Series   []BalanceSeries `json:"series"`
	Currency *string         `json:"currency"`
	Decimals int             `json:"decimals"`
}

// BalanceComparison is the total balance now against the day before a period.
type BalanceComparison struct {
	Current  money.Money  `json:"current"`
	Previous *money.Money `json:"previous"`
	Currency *string      `json:"currency"`
	Decimals int          `json:"decimals"`
}
