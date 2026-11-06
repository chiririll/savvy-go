package dto

import (
	"encoding/json"

	"savvy-go/internal/recurring"
)

// Recurring is a rule. Template is what it creates, in the shape the API takes
// to create a transaction (snake_case keys), plus a payment without an amount
// that each occurrence fills in; Schedule is when.
type Recurring struct {
	ID          int64              `json:"id"`
	Template    json.RawMessage    `json:"template"`
	Schedule    recurring.Schedule `json:"schedule"`
	IsActive    bool               `json:"isActive"`
	Error       *string            `json:"error"`
	NextRunDate *string            `json:"nextRunDate"`
	LastRunDate *string            `json:"lastRunDate"`
}

func NewRecurring(r recurring.Rule) Recurring {
	return Recurring{
		ID: r.ID, Template: r.Template, Schedule: r.Schedule, IsActive: r.IsActive, Error: r.Error,
		NextRunDate: r.NextRunDate, LastRunDate: r.LastRunDate,
	}
}
