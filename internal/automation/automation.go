package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	repo "savvy-go/internal/db/repo/automation"
	"savvy-go/internal/ledger"
	"savvy-go/internal/store"
)

type Rule struct {
	ID             int64
	Name           string
	Description    *string
	TriggerType    string
	Priority       int
	Conditions     map[string]any
	Actions        []map[string]any
	IsActive       bool
	StopProcessing bool
	RunsCount      int
	LastRunAt      *time.Time
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
}

type Log struct {
	ID                int64
	RuleID            int64
	TriggerEntityType *string
	TriggerEntityID   *int64
	ActionsExecuted   any
	Status            string
	ErrorMessage      *string
	CreatedAt         time.Time
}

type Input struct {
	Name           string
	Description    *string
	TriggerType    string
	Priority       int
	Conditions     map[string]any
	Actions        []map[string]any
	IsActive       *bool
	StopProcessing *bool
}

type Service struct {
	DB  store.DB
	Txs ledger.Transactions
}

func (s Service) repo() repo.Repo { return repo.Repo{DB: s.DB} }

func (s Service) All(ctx context.Context) ([]Rule, error) {
	return s.list(ctx, repo.Filter{})
}

func (s Service) ByID(ctx context.Context, id int64) (*Rule, error) {
	list, err := s.list(ctx, repo.Filter{ID: id})
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func (s Service) Create(ctx context.Context, in Input) (*Rule, error) {
	if in.Priority < 1 {
		in.Priority = 50
	}
	active := true
	if in.IsActive != nil {
		active = *in.IsActive
	}
	stop := false
	if in.StopProcessing != nil {
		stop = *in.StopProcessing
	}
	cond, _ := json.Marshal(normalizeConditions(in.Conditions))
	acts, _ := json.Marshal(in.Actions)
	if in.Actions == nil {
		acts = []byte("[]")
	}
	id, err := s.repo().Insert(ctx, repo.Fields{
		Name: in.Name, Description: in.Description, TriggerType: in.TriggerType, Priority: in.Priority,
		Conditions: string(cond), Actions: string(acts), IsActive: active, StopProcessing: stop,
	}, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Service) Update(ctx context.Context, id int64, in Input) (*Rule, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if in.Name == "" {
		in.Name = cur.Name
	}
	if in.TriggerType == "" {
		in.TriggerType = cur.TriggerType
	}
	if in.Priority < 1 {
		in.Priority = cur.Priority
	}
	if in.Conditions == nil {
		in.Conditions = cur.Conditions
	}
	if in.Actions == nil {
		in.Actions = cur.Actions
	}
	active := cur.IsActive
	if in.IsActive != nil {
		active = *in.IsActive
	}
	stop := cur.StopProcessing
	if in.StopProcessing != nil {
		stop = *in.StopProcessing
	}
	cond, _ := json.Marshal(normalizeConditions(in.Conditions))
	acts, _ := json.Marshal(in.Actions)
	err = s.repo().Update(ctx, id, repo.Fields{
		Name: in.Name, Description: in.Description, TriggerType: in.TriggerType, Priority: in.Priority,
		Conditions: string(cond), Actions: string(acts), IsActive: active, StopProcessing: stop,
	}, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Service) Delete(ctx context.Context, id int64) error {
	return s.repo().Delete(ctx, id)
}

func (s Service) Toggle(ctx context.Context, id int64) (*Rule, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	err = s.repo().SetActive(ctx, id, !cur.IsActive, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Service) Reorder(ctx context.Context, rules []struct {
	ID       int64 `json:"id"`
	Priority int   `json:"priority"`
}) error {
	for _, r := range rules {
		if err := s.repo().SetPriority(ctx, r.ID, r.Priority); err != nil {
			return err
		}
	}
	return nil
}

func (s Service) Logs(ctx context.Context, ruleID int64) ([]Log, error) {
	rows, err := s.repo().Logs(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	out := make([]Log, len(rows))
	for i, r := range rows {
		l := Log{ID: r.ID, RuleID: r.RuleID, Status: r.Status, TriggerEntityType: r.TriggerEntityType,
			TriggerEntityID: r.TriggerEntityID, ErrorMessage: r.ErrorMessage, CreatedAt: r.CreatedAt}
		if r.ActionsExecuted != "" {
			var v any
			_ = json.Unmarshal([]byte(r.ActionsExecuted), &v)
			l.ActionsExecuted = v
		}
		out[i] = l
	}
	return out, nil
}

func (s Service) Test(ctx context.Context, ruleID, txID int64) (map[string]any, error) {
	rule, err := s.ByID(ctx, ruleID)
	if err != nil || rule == nil {
		return nil, err
	}
	tx, err := s.Txs.ByID(ctx, txID)
	if err != nil || tx == nil {
		return nil, fmt.Errorf("transaction not found")
	}
	match := evaluateConditions(rule.Conditions, tx)
	return map[string]any{
		"conditionsMatch": match,
		"actions":         rule.Actions,
	}, nil
}

func (s Service) Process(ctx context.Context, trigger string, tx *ledger.Transaction) {
	if tx == nil || tx.Status != "confirmed" {
		return
	}
	rules, err := s.list(ctx, repo.Filter{ActiveOnly: true, TriggerType: trigger})
	if err != nil {
		slog.Error("automation load", "err", err)
		return
	}
	for i := range rules {
		rule := &rules[i]
		if !evaluateConditions(rule.Conditions, tx) {
			continue
		}
		results, err := s.executeActions(ctx, rule.Actions, tx)
		if err != nil {
			s.log(ctx, rule, tx, "error", results, err.Error())
			slog.Error("automation rule failed", "rule_id", rule.ID, "err", err)
			continue
		}
		s.log(ctx, rule, tx, "success", results, "")
		s.incrementRuns(ctx, rule.ID)
		if rule.StopProcessing {
			break
		}
	}
}

func (s Service) incrementRuns(ctx context.Context, id int64) {
	_ = s.repo().CountRun(ctx, id, time.Now().UTC())
}

func (s Service) log(
	ctx context.Context,
	rule *Rule,
	tx *ledger.Transaction,
	status string,
	actions any,
	errMsg string,
) {
	var actionsJSON string
	if actions != nil {
		raw, _ := json.Marshal(actions)
		actionsJSON = string(raw)
	}
	_ = s.repo().AddLog(ctx, rule.ID, tx.ID, status, actionsJSON, errMsg, time.Now().UTC())
}

func (s Service) executeActions(
	ctx context.Context,
	actions []map[string]any,
	tx *ledger.Transaction,
) ([]map[string]any, error) {
	var results []map[string]any
	for _, action := range actions {
		typ, _ := action["type"].(string)
		var result any
		var err error
		switch typ {
		case "set_category":
			result, err = s.actionSetCategory(ctx, action, tx)
		case "add_tags":
			result, err = s.actionAddTags(ctx, action, tx)
		case "remove_tags":
			result, err = s.actionRemoveTags(ctx, action, tx)
		case "set_description":
			result, err = s.actionSetDescription(ctx, action, tx)
		case "create_transfer":
			result, err = s.actionCreateTransfer(ctx, action, tx)
		}
		if err != nil {
			return results, err
		}
		results = append(results, map[string]any{"type": typ, "result": result})
		// Refresh tags/fields so later actions see updates.
		if fresh, e := s.Txs.ByID(ctx, tx.ID); e == nil && fresh != nil {
			*tx = *fresh
		}
	}
	return results, nil
}

func (s Service) actionSetCategory(ctx context.Context, action map[string]any, tx *ledger.Transaction) (any, error) {
	id, ok := asInt64(action["category_id"])
	if !ok {
		return false, nil
	}
	err := s.Txs.SetCategory(ctx, tx.ID, id)
	return err == nil, err
}

func (s Service) actionAddTags(ctx context.Context, action map[string]any, tx *ledger.Transaction) (any, error) {
	add := asInt64Slice(action["tag_ids"])
	if len(add) == 0 {
		return false, nil
	}
	seen := map[int64]bool{}
	var ids []int64
	for _, t := range tx.Tags {
		if !seen[t.ID] {
			seen[t.ID] = true
			ids = append(ids, t.ID)
		}
	}
	for _, id := range add {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return true, s.Txs.SetTags(ctx, tx.ID, ids)
}

func (s Service) actionRemoveTags(ctx context.Context, action map[string]any, tx *ledger.Transaction) (any, error) {
	remove := map[int64]bool{}
	for _, id := range asInt64Slice(action["tag_ids"]) {
		remove[id] = true
	}
	if len(remove) == 0 {
		return false, nil
	}
	var ids []int64
	for _, t := range tx.Tags {
		if !remove[t.ID] {
			ids = append(ids, t.ID)
		}
	}
	return true, s.Txs.SetTags(ctx, tx.ID, ids)
}

func (s Service) actionSetDescription(ctx context.Context, action map[string]any, tx *ledger.Transaction) (any, error) {
	tmpl := firstString(action, "value", "template", "description")
	if tmpl == "" {
		return false, nil
	}
	desc := parseTemplate(tmpl, tx)
	err := s.Txs.SetDescription(ctx, tx.ID, desc)
	return err == nil, err
}

func (s Service) actionCreateTransfer(ctx context.Context, action map[string]any, tx *ledger.Transaction) (any, error) {
	fromRaw := resolveValue(action["from_account_id"], tx)
	fromID, ok := asInt64(fromRaw)
	toID, ok2 := asInt64(action["to_account_id"])
	formula := action["amount_formula"]
	if formula == nil {
		formula = action["amount"]
	}
	if !ok || !ok2 || formula == nil {
		return nil, nil
	}
	amount := evaluateFormula(formula, tx)
	if !amount.IsPositive() {
		return nil, nil
	}
	desc := firstString(action, "description")
	if desc == "" {
		desc = "Auto-transfer by automation rule"
	}
	desc = parseTemplate(desc, tx)
	today := ledger.Today()
	status := "confirmed"
	created, err := s.Txs.Create(ctx, ledger.TxInput{
		Type: "transfer", AccountID: fromID, ToAccountID: &toID, Amount: amount,
		Description: &desc, Date: &today, Status: &status,
	})
	if err != nil || created == nil {
		return nil, err
	}
	return created.ID, nil
}

func (s Service) list(ctx context.Context, f repo.Filter) ([]Rule, error) {
	rows, err := s.repo().List(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]Rule, len(rows))
	for i, r := range rows {
		out[i] = automationFrom(r)
	}
	return out, nil
}

func automationFrom(row repo.Rule) Rule {
	r := Rule{
		ID: row.ID, Name: row.Name, Description: row.Description, TriggerType: row.TriggerType, Priority: row.Priority,
		RunsCount: row.RunsCount, IsActive: row.IsActive, StopProcessing: row.StopProcessing,
		LastRunAt: row.LastRunAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if row.Conditions != "" {
		_ = json.Unmarshal([]byte(row.Conditions), &r.Conditions)
	}
	if r.Conditions == nil {
		r.Conditions = map[string]any{}
	}
	if row.Actions != "" {
		_ = json.Unmarshal([]byte(row.Actions), &r.Actions)
	}
	return r
}
