// Package automationrepo stores automation rules and the log of their runs in
// a space database. Conditions and actions are JSON text here; package
// automation gives them meaning.
package automation

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

type Repo struct{ DB store.DB }

func (r Repo) q() *sqlc.Queries { return db.Q(r.DB) }

// Rule is a stored rule.
type Rule struct {
	ID                              int64
	Name, TriggerType               string
	Description                     *string
	Priority, RunsCount             int
	Conditions, Actions             string
	IsActive, StopProcessing        bool
	LastRunAt, CreatedAt, UpdatedAt *time.Time
}

// Fields is the writable part of a rule.
type Fields struct {
	Name, TriggerType        string
	Description              *string
	Priority                 int
	Conditions, Actions      string
	IsActive, StopProcessing bool
}

// Filter narrows List; the zero value lists every rule.
type Filter struct {
	ID          int64 // 0: any
	ActiveOnly  bool
	TriggerType string
}

func ruleOf(x sqlc.AutomationRule) Rule {
	r := Rule{
		ID: x.ID, Name: x.Name, TriggerType: x.TriggerType, Priority: int(x.Priority), RunsCount: int(x.RunsCount),
		Conditions: x.Conditions, Actions: x.Actions, IsActive: x.IsActive != 0, StopProcessing: x.StopProcessing != 0,
	}
	if x.Description.Valid {
		r.Description = &x.Description.String
	}
	for _, f := range []struct {
		from db.NString
		to   **time.Time
	}{{x.LastRunAt, &r.LastRunAt}, {x.CreatedAt, &r.CreatedAt}, {x.UpdatedAt, &r.UpdatedAt}} {
		if t, ok := db.ParseTime(f.from); ok {
			*f.to = &t
		}
	}
	return r
}

func (r Repo) List(ctx context.Context, f Filter) ([]Rule, error) {
	arg := sqlc.ListAutomationRulesParams{TriggerType: db.NullStringVal(f.TriggerType)}
	if f.ID != 0 {
		arg.ID = db.NI(f.ID)
	}
	if f.ActiveOnly {
		arg.ActiveOnly = db.Flag(true)
	}
	rows, err := r.q().ListAutomationRules(ctx, arg)
	if err != nil {
		return nil, err
	}
	out := make([]Rule, len(rows))
	for i, x := range rows {
		out[i] = ruleOf(x)
	}
	return out, nil
}

func (r Repo) Insert(ctx context.Context, f Fields, now time.Time) (int64, error) {
	res, err := r.q().InsertAutomationRule(ctx, sqlc.InsertAutomationRuleParams{
		Name: f.Name, Description: db.NullString(f.Description), TriggerType: f.TriggerType,
		Priority: int64(f.Priority), Conditions: f.Conditions, Actions: f.Actions,
		IsActive: db.BoolInt(f.IsActive), StopProcessing: db.BoolInt(f.StopProcessing),
		CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r Repo) Update(ctx context.Context, id int64, f Fields, now time.Time) error {
	return r.q().UpdateAutomationRule(ctx, sqlc.UpdateAutomationRuleParams{
		Name: f.Name, Description: db.NullString(f.Description), TriggerType: f.TriggerType,
		Priority: int64(f.Priority), Conditions: f.Conditions, Actions: f.Actions,
		IsActive: db.BoolInt(f.IsActive), StopProcessing: db.BoolInt(f.StopProcessing),
		UpdatedAt: db.TimeNS(now), ID: id,
	})
}

func (r Repo) Delete(ctx context.Context, id int64) error { return r.q().DeleteAutomationRule(ctx, id) }

func (r Repo) SetActive(ctx context.Context, id int64, active bool, now time.Time) error {
	return r.q().
		ToggleAutomationRule(ctx,
			sqlc.ToggleAutomationRuleParams{IsActive: db.BoolInt(active), UpdatedAt: db.TimeNS(now), ID: id})
}

func (r Repo) SetPriority(ctx context.Context, id int64, priority int) error {
	return r.q().SetAutomationPriority(ctx, sqlc.SetAutomationPriorityParams{Priority: int64(priority), ID: id})
}

// CountRun records that a rule ran.
func (r Repo) CountRun(ctx context.Context, id int64, now time.Time) error {
	return r.q().
		IncrementAutomationRuns(ctx,
			sqlc.IncrementAutomationRunsParams{LastRunAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now), ID: id})
}

// Log is one run of a rule.
type Log struct {
	ID, RuleID                      int64
	TriggerEntityType, ErrorMessage *string
	TriggerEntityID                 *int64
	ActionsExecuted                 string // JSON text, empty when none
	Status                          string
	CreatedAt                       time.Time
}

func (r Repo) Logs(ctx context.Context, ruleID int64) ([]Log, error) {
	rows, err := r.q().ListAutomationLogs(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	out := make([]Log, len(rows))
	for i, x := range rows {
		l := Log{ID: x.ID, RuleID: x.RuleID, Status: x.Status, ActionsExecuted: x.ActionsExecuted.String}
		if x.TriggerEntityType.Valid {
			l.TriggerEntityType = &x.TriggerEntityType.String
		}
		if x.TriggerEntityID.Valid {
			l.TriggerEntityID = &x.TriggerEntityID.Int64
		}
		if x.ErrorMessage.Valid {
			l.ErrorMessage = &x.ErrorMessage.String
		}
		l.CreatedAt, _ = db.ParseTimeString(x.CreatedAt)
		out[i] = l
	}
	return out, nil
}

// AddLog records a run of a rule on a transaction.
func (r Repo) AddLog(ctx context.Context, ruleID, txID int64, status, actionsJSON, errMsg string, now time.Time) error {
	return r.q().InsertAutomationLog(ctx, sqlc.InsertAutomationLogParams{
		RuleID: ruleID, TriggerEntityType: db.NS("Transaction"), TriggerEntityID: db.NI(txID),
		ActionsExecuted: db.NullStringVal(actionsJSON), Status: status, ErrorMessage: db.NullStringVal(errMsg),
		CreatedAt: db.FormatTime(now),
	})
}
