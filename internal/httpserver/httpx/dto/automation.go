package dto

import (
	"time"

	"savvy-go/internal/automation"
)

type AutomationRule struct {
	ID             int64            `json:"id"`
	Name           string           `json:"name"`
	Description    *string          `json:"description"`
	TriggerType    string           `json:"triggerType"`
	Priority       int              `json:"priority"`
	Conditions     map[string]any   `json:"conditions"`
	Actions        []map[string]any `json:"actions"`
	IsActive       bool             `json:"isActive"`
	StopProcessing bool             `json:"stopProcessing"`
	RunsCount      int              `json:"runsCount"`
	LastRunAt      *time.Time       `json:"lastRunAt"`
	CreatedAt      *time.Time       `json:"createdAt"`
	UpdatedAt      *time.Time       `json:"updatedAt"`
}

func NewAutomationRule(r automation.Rule) AutomationRule {
	return AutomationRule{
		ID: r.ID, Name: r.Name, Description: r.Description, TriggerType: r.TriggerType, Priority: r.Priority,
		Conditions: r.Conditions, Actions: r.Actions, IsActive: r.IsActive, StopProcessing: r.StopProcessing,
		RunsCount: r.RunsCount, LastRunAt: utc(r.LastRunAt), CreatedAt: utc(r.CreatedAt), UpdatedAt: utc(r.UpdatedAt),
	}
}

type AutomationLog struct {
	ID                int64     `json:"id"`
	RuleID            int64     `json:"ruleId"`
	TriggerEntityType *string   `json:"triggerEntityType"`
	TriggerEntityID   *int64    `json:"triggerEntityId"`
	ActionsExecuted   any       `json:"actionsExecuted"`
	Status            string    `json:"status"`
	ErrorMessage      *string   `json:"errorMessage"`
	CreatedAt         time.Time `json:"createdAt"`
}

func NewAutomationLog(l automation.Log) AutomationLog {
	return AutomationLog{
		ID: l.ID, RuleID: l.RuleID, TriggerEntityType: l.TriggerEntityType, TriggerEntityID: l.TriggerEntityID,
		ActionsExecuted: l.ActionsExecuted, Status: l.Status, ErrorMessage: l.ErrorMessage, CreatedAt: l.CreatedAt.UTC(),
	}
}
