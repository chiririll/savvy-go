package dto

import (
	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
)

type BudgetProgress struct {
	Spent       money.Money `json:"spent"`
	Remaining   money.Money `json:"remaining"`
	Percent     float64     `json:"percent"`
	PeriodStart string      `json:"periodStart"`
	PeriodEnd   string      `json:"periodEnd"`
	IsExceeded  bool        `json:"isExceeded"`
}

func NewBudgetProgress(p ledger.BudgetProgress) BudgetProgress {
	return BudgetProgress{
		Spent: p.Spent, Remaining: p.Remaining, Percent: p.Percent,
		PeriodStart: p.PeriodStart, PeriodEnd: p.PeriodEnd, IsExceeded: p.IsExceeded,
	}
}

type Budget struct {
	ID              int64           `json:"id"`
	Name            string          `json:"name"`
	Amount          money.Money     `json:"amount"`
	Currency        *Currency       `json:"currency"`
	Period          string          `json:"period"`
	StartDate       *string         `json:"startDate"`
	EndDate         *string         `json:"endDate"`
	IsGlobal        bool            `json:"isGlobal"`
	NotifyAtPercent *int            `json:"notifyAtPercent"`
	IsActive        bool            `json:"isActive"`
	Categories      []Category      `json:"categories"`
	Tags            []Tag           `json:"tags"`
	Progress        *BudgetProgress `json:"progress"`
}

func NewBudget(b ledger.Budget) Budget {
	return Budget{
		ID: b.ID, Name: b.Name, Amount: b.Amount, Currency: ptr(b.Currency, NewCurrency),
		Period: b.Period, StartDate: b.StartDate, EndDate: b.EndDate,
		IsGlobal: b.IsGlobal, NotifyAtPercent: b.NotifyAtPercent, IsActive: b.IsActive,
		Categories: Map(b.Categories, NewCategory),
		Tags:       Map(b.Tags, NewTag),
		Progress:   ptr(b.Progress, NewBudgetProgress),
	}
}
