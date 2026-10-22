package dto

import (
	"savvy-go/internal/money"
	"savvy-go/internal/recurring"
)

type Recurring struct {
	ID          int64        `json:"id"`
	Type        string       `json:"type"`
	Amount      money.Money  `json:"amount"`
	ToAmount    *money.Money `json:"toAmount"`
	IsEstimated bool         `json:"isEstimated"`
	Description *string      `json:"description"`
	Frequency   string       `json:"frequency"`
	Interval    int          `json:"interval"`
	DayOfWeek   *int         `json:"dayOfWeek"`
	DayOfMonth  *int         `json:"dayOfMonth"`
	StartDate   string       `json:"startDate"`
	EndDate     *string      `json:"endDate"`
	NextRunDate string       `json:"nextRunDate"`
	LastRunDate *string      `json:"lastRunDate"`
	IsActive    bool         `json:"isActive"`
	Account     *Account     `json:"account"`
	ToAccount   *Account     `json:"toAccount"`
	Category    *Category    `json:"category"`
	Tags        []Tag        `json:"tags"`
}

func NewRecurring(r recurring.Recurring) Recurring {
	return Recurring{
		ID: r.ID, Type: r.Type, Amount: r.Amount, ToAmount: r.ToAmount, IsEstimated: r.IsEstimated,
		Description: r.Description,
		Frequency:   r.Frequency, Interval: r.Interval, DayOfWeek: r.DayOfWeek, DayOfMonth: r.DayOfMonth,
		StartDate: r.StartDate, EndDate: r.EndDate, NextRunDate: r.NextRunDate, LastRunDate: r.LastRunDate,
		IsActive:  r.IsActive,
		Account:   ptr(r.Account, NewAccount),
		ToAccount: ptr(r.ToAccount, NewAccount),
		Category:  ptr(r.Category, NewCategory),
		Tags:      Map(r.Tags, NewTag),
	}
}
