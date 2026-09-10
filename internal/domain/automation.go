package domain

import (
	"context"
	"database/sql"
	"encoding/json"

	"fmt"
	"log/slog"
	"math"
	"regexp"
	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type AutomationRule struct {
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

func triggerLabel(t string) (string, string) {
	switch t {
	case "on_transaction_create":
		return "On Transaction Create", "Triggers when a new transaction is created"
	case "on_transaction_update":
		return "On Transaction Update", "Triggers when a transaction is updated"
	default:
		return t, ""
	}
}

func (r AutomationRule) JSON() map[string]any {
	label, _ := triggerLabel(r.TriggerType)
	m := map[string]any{
		"id": r.ID, "name": r.Name, "description": r.Description,
		"trigger_type": r.TriggerType, "trigger_label": label,
		"priority": r.Priority, "conditions": r.Conditions, "actions": r.Actions,
		"is_active": r.IsActive, "stop_processing": r.StopProcessing,
		"runs_count": r.RunsCount,
	}
	if r.LastRunAt != nil {
		m["last_run_at"] = r.LastRunAt.UTC().Format(time.RFC3339Nano)
	} else {
		m["last_run_at"] = nil
	}
	if r.CreatedAt != nil {
		m["created_at"] = r.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	if r.UpdatedAt != nil {
		m["updated_at"] = r.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	return m
}

type AutomationLog struct {
	ID                int64
	RuleID            int64
	TriggerEntityType *string
	TriggerEntityID   *int64
	ActionsExecuted   any
	Status            string
	ErrorMessage      *string
	CreatedAt         time.Time
}

func (l AutomationLog) JSON() map[string]any {
	return map[string]any{
		"id": l.ID, "rule_id": l.RuleID,
		"trigger_entity_type": l.TriggerEntityType, "trigger_entity_id": l.TriggerEntityID,
		"actions_executed": l.ActionsExecuted, "status": l.Status,
		"error_message": l.ErrorMessage,
		"created_at":    l.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

type AutomationInput struct {
	Name           string
	Description    *string
	TriggerType    string
	Priority       int
	Conditions     map[string]any
	Actions        []map[string]any
	IsActive       *bool
	StopProcessing *bool
}

type Automation struct {
	DB  *sql.DB
	Txs Transactions
}

func (s Automation) All(ctx context.Context) ([]AutomationRule, error) {
	return s.list(ctx, sqlc.ListAutomationRulesParams{})
}

func (s Automation) ByID(ctx context.Context, id int64) (*AutomationRule, error) {
	list, err := s.list(ctx, sqlc.ListAutomationRulesParams{ID: db.NI(id)})
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func (s Automation) Create(ctx context.Context, in AutomationInput) (*AutomationRule, error) {
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
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Q(s.DB).InsertAutomationRule(ctx, sqlc.InsertAutomationRuleParams{
		Name: in.Name, Description: db.NullString(in.Description), TriggerType: in.TriggerType,
		Priority: int64(in.Priority), Conditions: string(cond), Actions: string(acts),
		IsActive: db.BoolInt(active), StopProcessing: db.BoolInt(stop), CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
	})
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

func (s Automation) Update(ctx context.Context, id int64, in AutomationInput) (*AutomationRule, error) {
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
	if in.Description == nil {
		in.Description = cur.Description
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
	now := time.Now().UTC().Format(time.RFC3339)
	err = db.Q(s.DB).UpdateAutomationRule(ctx, sqlc.UpdateAutomationRuleParams{
		Name: in.Name, Description: db.NullString(in.Description), TriggerType: in.TriggerType,
		Priority: int64(in.Priority), Conditions: string(cond), Actions: string(acts),
		IsActive: db.BoolInt(active), StopProcessing: db.BoolInt(stop), UpdatedAt: db.NS(now), ID: id,
	})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Automation) Delete(ctx context.Context, id int64) error {
	return db.Q(s.DB).DeleteAutomationRule(ctx, id)
}

func (s Automation) Toggle(ctx context.Context, id int64) (*AutomationRule, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err = db.Q(s.DB).ToggleAutomationRule(ctx, sqlc.ToggleAutomationRuleParams{IsActive: db.BoolInt(!cur.IsActive), UpdatedAt: db.NS(now), ID: id})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Automation) Reorder(ctx context.Context, rules []struct {
	ID       int64 `json:"id"`
	Priority int   `json:"priority"`
}) error {
	for _, r := range rules {
		if err := db.Q(s.DB).SetAutomationPriority(ctx, sqlc.SetAutomationPriorityParams{Priority: int64(r.Priority), ID: r.ID}); err != nil {
			return err
		}
	}
	return nil
}

func (s Automation) Logs(ctx context.Context, ruleID int64) ([]AutomationLog, error) {
	rows, err := db.Q(s.DB).ListAutomationLogs(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	out := make([]AutomationLog, 0, len(rows))
	for _, r := range rows {
		l := AutomationLog{ID: r.ID, RuleID: r.RuleID, Status: r.Status}
		if r.TriggerEntityType.Valid {
			l.TriggerEntityType = &r.TriggerEntityType.String
		}
		if r.TriggerEntityID.Valid {
			l.TriggerEntityID = &r.TriggerEntityID.Int64
		}
		if r.ActionsExecuted.Valid && r.ActionsExecuted.String != "" {
			var v any
			_ = json.Unmarshal([]byte(r.ActionsExecuted.String), &v)
			l.ActionsExecuted = v
		}
		if r.ErrorMessage.Valid {
			l.ErrorMessage = &r.ErrorMessage.String
		}
		if tm, ok := parseNullTime(sql.NullString{String: r.CreatedAt, Valid: true}); ok {
			l.CreatedAt = tm
		}
		out = append(out, l)
	}
	return out, nil
}

func (s Automation) Test(ctx context.Context, ruleID, txID int64) (map[string]any, error) {
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
		"conditions_match": match,
		"would_execute":    match,
		"actions":          rule.Actions,
	}, nil
}

func (s Automation) Process(ctx context.Context, trigger string, tx *Transaction) {
	if tx == nil || tx.Status != "confirmed" {
		return
	}
	rules, err := s.list(ctx, sqlc.ListAutomationRulesParams{ActiveOnly: db.Flag(true), TriggerType: db.NullStringVal(trigger)})
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

func (s Automation) incrementRuns(ctx context.Context, id int64) {
	now := time.Now().UTC().Format(time.RFC3339)
	_ = db.Q(s.DB).IncrementAutomationRuns(ctx, sqlc.IncrementAutomationRunsParams{LastRunAt: db.NS(now), UpdatedAt: db.NS(now), ID: id})
}

func (s Automation) log(ctx context.Context, rule *AutomationRule, tx *Transaction, status string, actions any, errMsg string) {
	now := time.Now().UTC().Format(time.RFC3339)
	var actsS sql.NullString
	if actions != nil {
		raw, _ := json.Marshal(actions)
		actsS = db.NS(string(raw))
	}
	_ = db.Q(s.DB).InsertAutomationLog(ctx, sqlc.InsertAutomationLogParams{
		RuleID: rule.ID, TriggerEntityType: db.NS("Transaction"), TriggerEntityID: db.NI(tx.ID),
		ActionsExecuted: actsS, Status: status, ErrorMessage: db.NullStringVal(errMsg), CreatedAt: now,
	})
}

func (s Automation) executeActions(ctx context.Context, actions []map[string]any, tx *Transaction) ([]map[string]any, error) {
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

func (s Automation) actionSetCategory(ctx context.Context, action map[string]any, tx *Transaction) (any, error) {
	id, ok := asInt64(action["category_id"])
	if !ok {
		return false, nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err := db.Q(s.DB).UpdateTransactionCategory(ctx, sqlc.UpdateTransactionCategoryParams{CategoryID: db.NI(id), UpdatedAt: db.NS(now), ID: tx.ID})
	return err == nil, err
}

func (s Automation) actionAddTags(ctx context.Context, action map[string]any, tx *Transaction) (any, error) {
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
	return true, s.Txs.saveTags(ctx, tx.ID, ids)
}

func (s Automation) actionRemoveTags(ctx context.Context, action map[string]any, tx *Transaction) (any, error) {
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
	return true, s.Txs.saveTags(ctx, tx.ID, ids)
}

func (s Automation) actionSetDescription(ctx context.Context, action map[string]any, tx *Transaction) (any, error) {
	tmpl := firstString(action, "value", "template", "description")
	if tmpl == "" {
		return false, nil
	}
	desc := parseTemplate(tmpl, tx)
	now := time.Now().UTC().Format(time.RFC3339)
	err := db.Q(s.DB).UpdateTransactionDescription(ctx, sqlc.UpdateTransactionDescriptionParams{Description: db.NS(desc), UpdatedAt: db.NS(now), ID: tx.ID})
	return err == nil, err
}

func (s Automation) actionCreateTransfer(ctx context.Context, action map[string]any, tx *Transaction) (any, error) {
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
	if amount <= 0 {
		return nil, nil
	}
	desc := firstString(action, "description")
	if desc == "" {
		desc = "Auto-transfer by automation rule"
	}
	desc = parseTemplate(desc, tx)
	today := time.Now().UTC().Format("2006-01-02")
	status := "confirmed"
	created, err := s.Txs.Create(ctx, TxInput{
		Type: "transfer", AccountID: fromID, ToAccountID: &toID, Amount: amount,
		Description: &desc, Date: &today, Status: &status,
	})
	if err != nil || created == nil {
		return nil, err
	}
	return created.ID, nil
}

func (s Automation) list(ctx context.Context, arg sqlc.ListAutomationRulesParams) ([]AutomationRule, error) {
	rows, err := db.Q(s.DB).ListAutomationRules(ctx, arg)
	if err != nil {
		return nil, err
	}
	out := make([]AutomationRule, 0, len(rows))
	for _, r := range rows {
		out = append(out, automationFrom(r))
	}
	return out, nil
}

func automationFrom(row sqlc.AutomationRule) AutomationRule {
	r := AutomationRule{
		ID: row.ID, Name: row.Name, TriggerType: row.TriggerType, Priority: int(row.Priority),
		RunsCount: int(row.RunsCount), IsActive: row.IsActive != 0, StopProcessing: row.StopProcessing != 0,
	}
	if row.Description.Valid {
		r.Description = &row.Description.String
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
	if tm, ok := parseNullTime(row.LastRunAt); ok {
		r.LastRunAt = &tm
	}
	if tm, ok := parseNullTime(row.CreatedAt); ok {
		r.CreatedAt = &tm
	}
	if tm, ok := parseNullTime(row.UpdatedAt); ok {
		r.UpdatedAt = &tm
	}
	return r
}

func normalizeConditions(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{"match": "all", "conditions": []any{}}
	}
	conds, _ := in["conditions"].([]any)
	for i, c := range conds {
		m, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if _, has := m["op"]; !has {
			if op, ok := m["operator"]; ok {
				m["op"] = op
				delete(m, "operator")
			}
		}
		conds[i] = m
	}
	in["conditions"] = conds
	if _, ok := in["match"]; !ok {
		in["match"] = "all"
	}
	return in
}

func evaluateConditions(group map[string]any, tx *Transaction) bool {
	if group == nil {
		return true
	}
	match, _ := group["match"].(string)
	if match == "" {
		match = "all"
	}
	raw, _ := group["conditions"].([]any)
	if len(raw) == 0 {
		return true
	}
	for _, item := range raw {
		cond, _ := item.(map[string]any)
		ok := evaluateCondition(cond, tx)
		if match == "any" && ok {
			return true
		}
		if match == "all" && !ok {
			return false
		}
	}
	return match == "all"
}

func evaluateCondition(cond map[string]any, tx *Transaction) bool {
	field, _ := cond["field"].(string)
	op, _ := cond["op"].(string)
	if op == "" {
		op, _ = cond["operator"].(string)
	}
	if op == "" {
		op = "equals"
	}
	value := cond["value"]
	entity := fieldValue(tx, field)
	switch op {
	case "equals":
		return valuesEqual(entity, value)
	case "not_equals":
		return !valuesEqual(entity, value)
	case "in":
		return inSlice(entity, value)
	case "not_in":
		return !inSlice(entity, value)
	case "gt", "gte", "lt", "lte":
		a, ok1 := asFloat(entity)
		b, ok2 := asFloat(value)
		if !ok1 || !ok2 {
			return false
		}
		switch op {
		case "gt":
			return a > b
		case "gte":
			return a >= b
		case "lt":
			return a < b
		case "lte":
			return a <= b
		}
	case "between":
		arr, _ := value.([]any)
		if len(arr) != 2 {
			return false
		}
		v, ok := asFloat(entity)
		lo, ok1 := asFloat(arr[0])
		hi, ok2 := asFloat(arr[1])
		return ok && ok1 && ok2 && v >= lo && v <= hi
	case "contains":
		return strings.Contains(strings.ToLower(asString(entity)), strings.ToLower(asString(value)))
	case "not_contains":
		return !strings.Contains(strings.ToLower(asString(entity)), strings.ToLower(asString(value)))
	case "starts_with":
		return strings.HasPrefix(strings.ToLower(asString(entity)), strings.ToLower(asString(value)))
	case "ends_with":
		return strings.HasSuffix(strings.ToLower(asString(entity)), strings.ToLower(asString(value)))
	case "matches":
		re, err := regexp.Compile("(?i)" + asString(value))
		return err == nil && re.MatchString(asString(entity))
	case "is_null":
		return entity == nil || asString(entity) == ""
	case "is_not_null":
		return entity != nil && asString(entity) != ""
	case "has_any":
		return tagIntersect(tx, value) > 0
	case "has_all":
		want := asInt64Slice(value)
		have := tagIDSet(tx)
		for _, id := range want {
			if !have[id] {
				return false
			}
		}
		return true
	case "has_none":
		return tagIntersect(tx, value) == 0
	}
	return false
}

func fieldValue(tx *Transaction, field string) any {
	if strings.Contains(field, ".") {
		parts := strings.Split(field, ".")
		var cur any = tx
		for _, p := range parts {
			cur = stepField(cur, p)
			if cur == nil {
				return nil
			}
		}
		return cur
	}
	return stepField(tx, field)
}

func stepField(cur any, field string) any {
	switch v := cur.(type) {
	case *Transaction:
		switch field {
		case "type":
			return v.Type
		case "amount":
			return v.Amount
		case "description":
			if v.Description == nil {
				return nil
			}
			return *v.Description
		case "date":
			if v.Date == nil {
				return nil
			}
			return *v.Date
		case "status":
			return v.Status
		case "account_id":
			return v.AccountID
		case "to_account_id":
			if v.ToAccountID == nil {
				return nil
			}
			return *v.ToAccountID
		case "category_id":
			if v.CategoryID == nil {
				return nil
			}
			return *v.CategoryID
		case "account":
			return v.Account
		case "toAccount", "to_account":
			return v.ToAccount
		case "category":
			return v.Category
		case "tags":
			return v.Tags
		}
	case *Account:
		switch field {
		case "id":
			return v.ID
		case "name":
			return v.Name
		case "type":
			return v.Type
		}
	case *Category:
		switch field {
		case "id":
			return v.ID
		case "name":
			return v.Name
		case "type":
			return v.Type
		}
	}
	return nil
}

func valuesEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if fa, ok := asFloat(a); ok {
		if fb, ok := asFloat(b); ok {
			return fa == fb
		}
	}
	return asString(a) == asString(b)
}

func inSlice(entity, value any) bool {
	arr, ok := value.([]any)
	if !ok {
		return false
	}
	for _, item := range arr {
		if valuesEqual(entity, item) {
			return true
		}
	}
	return false
}

func tagIDSet(tx *Transaction) map[int64]bool {
	m := map[int64]bool{}
	for _, t := range tx.Tags {
		m[t.ID] = true
	}
	return m
}

func tagIntersect(tx *Transaction, value any) int {
	have := tagIDSet(tx)
	n := 0
	for _, id := range asInt64Slice(value) {
		if have[id] {
			n++
		}
	}
	return n
}

func resolveValue(value any, tx *Transaction) any {
	s, ok := value.(string)
	if !ok {
		return value
	}
	if strings.HasPrefix(s, "{{") && strings.HasSuffix(s, "}}") {
		path := strings.TrimSpace(s[2 : len(s)-2])
		path = strings.TrimPrefix(path, "transaction.")
		path = strings.TrimPrefix(path, "entity.")
		return fieldValue(tx, path)
	}
	return value
}

var placeholderRe = regexp.MustCompile(`\{\{(.+?)\}\}`)

func parseTemplate(tmpl string, tx *Transaction) string {
	return placeholderRe.ReplaceAllStringFunc(tmpl, func(m string) string {
		path := strings.TrimSpace(m[2 : len(m)-2])
		if strings.HasPrefix(path, "transaction.") || strings.HasPrefix(path, "entity.") {
			path = strings.TrimPrefix(path, "transaction.")
			path = strings.TrimPrefix(path, "entity.")
			return asString(fieldValue(tx, path))
		}
		return m
	})
}

func evaluateFormula(formula any, tx *Transaction) float64 {
	if f, ok := asFloat(formula); ok {
		return f
	}
	s, ok := formula.(string)
	if !ok {
		return 0
	}
	s = placeholderRe.ReplaceAllStringFunc(s, func(m string) string {
		path := strings.TrimSpace(m[2 : len(m)-2])
		if strings.HasPrefix(path, "transaction.") || strings.HasPrefix(path, "entity.") {
			path = strings.TrimPrefix(path, "transaction.")
			path = strings.TrimPrefix(path, "entity.")
			v := fieldValue(tx, path)
			if f, ok := asFloat(v); ok {
				return strconv.FormatFloat(f, 'f', -1, 64)
			}
			return "0"
		}
		return "0"
	})
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) || strings.ContainsRune("+*-/(). \t", r) || r == '.' {
			b.WriteRune(r)
		}
	}
	v, err := evalArith(b.String())
	if err != nil {
		return 0
	}
	return v
}

func evalArith(s string) (float64, error) {
	p := &arithParser{s: strings.TrimSpace(s)}
	v, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	p.skip()
	if p.pos < len(p.s) {
		return 0, fmt.Errorf("trailing")
	}
	return v, nil
}

type arithParser struct {
	s   string
	pos int
}

func (p *arithParser) skip() {
	for p.pos < len(p.s) && (p.s[p.pos] == ' ' || p.s[p.pos] == '\t') {
		p.pos++
	}
}

func (p *arithParser) parseExpr() (float64, error) {
	v, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for {
		p.skip()
		if p.pos >= len(p.s) {
			return v, nil
		}
		op := p.s[p.pos]
		if op != '+' && op != '-' {
			return v, nil
		}
		p.pos++
		r, err := p.parseTerm()
		if err != nil {
			return 0, err
		}
		if op == '+' {
			v += r
		} else {
			v -= r
		}
	}
}

func (p *arithParser) parseTerm() (float64, error) {
	v, err := p.parseFactor()
	if err != nil {
		return 0, err
	}
	for {
		p.skip()
		if p.pos >= len(p.s) {
			return v, nil
		}
		op := p.s[p.pos]
		if op != '*' && op != '/' {
			return v, nil
		}
		p.pos++
		r, err := p.parseFactor()
		if err != nil {
			return 0, err
		}
		if op == '*' {
			v *= r
		} else if r == 0 {
			return 0, fmt.Errorf("div0")
		} else {
			v /= r
		}
	}
}

func (p *arithParser) parseFactor() (float64, error) {
	p.skip()
	if p.pos < len(p.s) && p.s[p.pos] == '(' {
		p.pos++
		v, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		p.skip()
		if p.pos < len(p.s) && p.s[p.pos] == ')' {
			p.pos++
		}
		return v, nil
	}
	if p.pos < len(p.s) && (p.s[p.pos] == '+' || p.s[p.pos] == '-') {
		sign := 1.0
		if p.s[p.pos] == '-' {
			sign = -1
		}
		p.pos++
		v, err := p.parseFactor()
		return sign * v, err
	}
	start := p.pos
	for p.pos < len(p.s) && (unicode.IsDigit(rune(p.s[p.pos])) || p.s[p.pos] == '.') {
		p.pos++
	}
	if start == p.pos {
		return 0, fmt.Errorf("number")
	}
	return strconv.ParseFloat(p.s[start:p.pos], 64)
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(n, 64)
		return f, err == nil
	}
	return 0, false
}

func asInt64(v any) (int64, bool) {
	f, ok := asFloat(v)
	if !ok {
		return 0, false
	}
	return int64(math.Round(f)), true
}

func asInt64Slice(v any) []int64 {
	switch arr := v.(type) {
	case []any:
		var out []int64
		for _, item := range arr {
			if id, ok := asInt64(item); ok {
				out = append(out, id)
			}
		}
		return out
	case []int64:
		return arr
	}
	return nil
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case *string:
		if s == nil {
			return ""
		}
		return *s
	}
	return fmt.Sprint(v)
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok && s != "" {
			return s
		}
	}
	return ""
}
