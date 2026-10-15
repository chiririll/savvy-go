package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/shopspring/decimal"

	repo "savvy-go/internal/db/repo/automation"
	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
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

func (s Service) log(ctx context.Context, rule *Rule, tx *ledger.Transaction, status string, actions any, errMsg string) {
	var actionsJSON string
	if actions != nil {
		raw, _ := json.Marshal(actions)
		actionsJSON = string(raw)
	}
	_ = s.repo().AddLog(ctx, rule.ID, tx.ID, status, actionsJSON, errMsg, time.Now().UTC())
}

func (s Service) executeActions(ctx context.Context, actions []map[string]any, tx *ledger.Transaction) ([]map[string]any, error) {
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

func evaluateConditions(group map[string]any, tx *ledger.Transaction) bool {
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

func evaluateCondition(cond map[string]any, tx *ledger.Transaction) bool {
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
		return compareNumbers(op, entity, value)
	case "between":
		return betweenNumbers(entity, value)
	case "contains", "not_contains", "starts_with", "ends_with", "matches":
		return matchText(op, asString(entity), asString(value))
	case "is_null":
		return entity == nil || asString(entity) == ""
	case "is_not_null":
		return entity != nil && asString(entity) != ""
	case "has_any", "has_all", "has_none":
		return matchTags(op, tx, value)
	}
	return false
}

func compareNumbers(op string, entity, value any) bool {
	a, ok1 := asDecimal(entity)
	b, ok2 := asDecimal(value)
	if !ok1 || !ok2 {
		return false
	}
	switch op {
	case "gt":
		return a.GreaterThan(b)
	case "gte":
		return a.GreaterThanOrEqual(b)
	case "lt":
		return a.LessThan(b)
	case "lte":
		return a.LessThanOrEqual(b)
	}
	return false
}

func betweenNumbers(entity, value any) bool {
	arr, _ := value.([]any)
	if len(arr) != 2 {
		return false
	}
	v, ok := asDecimal(entity)
	lo, ok1 := asDecimal(arr[0])
	hi, ok2 := asDecimal(arr[1])
	return ok && ok1 && ok2 && v.GreaterThanOrEqual(lo) && v.LessThanOrEqual(hi)
}

// matchText compares case-insensitively; matches takes value as a regexp.
func matchText(op, entity, value string) bool {
	e, v := strings.ToLower(entity), strings.ToLower(value)
	switch op {
	case "contains":
		return strings.Contains(e, v)
	case "not_contains":
		return !strings.Contains(e, v)
	case "starts_with":
		return strings.HasPrefix(e, v)
	case "ends_with":
		return strings.HasSuffix(e, v)
	case "matches":
		re, err := regexp.Compile("(?i)" + value)
		return err == nil && re.MatchString(entity)
	}
	return false
}

func matchTags(op string, tx *ledger.Transaction, value any) bool {
	switch op {
	case "has_any":
		return tagIntersect(tx, value) > 0
	case "has_none":
		return tagIntersect(tx, value) == 0
	case "has_all":
		have := tagIDSet(tx)
		for _, id := range asInt64Slice(value) {
			if !have[id] {
				return false
			}
		}
		return true
	}
	return false
}

func fieldValue(tx *ledger.Transaction, field string) any {
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
	case *ledger.Transaction:
		return transactionField(v, field)
	case *ledger.Account:
		return namedField(v.ID, v.Name, v.Type, field)
	case *ledger.Category:
		return namedField(v.ID, v.Name, v.Type, field)
	}
	return nil
}

func namedField(id int64, name, typ, field string) any {
	switch field {
	case "id":
		return id
	case "name":
		return name
	case "type":
		return typ
	}
	return nil
}

func transactionField(v *ledger.Transaction, field string) any {
	switch field {
	case "type":
		return v.Type
	case "amount":
		return v.Amount
	case "description":
		return deref(v.Description)
	case "date":
		return deref(v.Date)
	case "status":
		return v.Status
	case "account_id":
		return v.AccountID
	case "to_account_id":
		return deref(v.ToAccountID)
	case "category_id":
		return deref(v.CategoryID)
	case "account":
		return v.Account
	case "toAccount", "to_account":
		return v.ToAccount
	case "category":
		return v.Category
	case "tags":
		return v.Tags
	}
	return nil
}

// deref returns the value p points to, or an untyped nil for a nil pointer.
func deref[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}

func valuesEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if da, ok := asDecimal(a); ok {
		if db, ok := asDecimal(b); ok {
			return da.Equal(db)
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

func tagIDSet(tx *ledger.Transaction) map[int64]bool {
	m := map[int64]bool{}
	for _, t := range tx.Tags {
		m[t.ID] = true
	}
	return m
}

func tagIntersect(tx *ledger.Transaction, value any) int {
	have := tagIDSet(tx)
	n := 0
	for _, id := range asInt64Slice(value) {
		if have[id] {
			n++
		}
	}
	return n
}

func resolveValue(value any, tx *ledger.Transaction) any {
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

func parseTemplate(tmpl string, tx *ledger.Transaction) string {
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

func evaluateFormula(formula any, tx *ledger.Transaction) decimal.Decimal {
	if d, ok := asDecimal(formula); ok {
		return d
	}
	str, ok := formula.(string)
	if !ok {
		return decimal.Zero
	}
	str = placeholderRe.ReplaceAllStringFunc(str, func(m string) string {
		path := strings.TrimSpace(m[2 : len(m)-2])
		if strings.HasPrefix(path, "transaction.") || strings.HasPrefix(path, "entity.") {
			path = strings.TrimPrefix(path, "transaction.")
			path = strings.TrimPrefix(path, "entity.")
			v := fieldValue(tx, path)
			if d, ok := asDecimal(v); ok {
				return d.String()
			}
			return "0"
		}
		return "0"
	})
	var b strings.Builder
	for _, r := range str {
		if unicode.IsDigit(r) || strings.ContainsRune("+*-/(). \t", r) || r == '.' {
			b.WriteRune(r)
		}
	}
	v, err := evalArith(b.String())
	if err != nil {
		return decimal.Zero
	}
	return v
}

func evalArith(s string) (decimal.Decimal, error) {
	p := &arithParser{s: strings.TrimSpace(s)}
	v, err := p.parseExpr()
	if err != nil {
		return decimal.Zero, err
	}
	p.skip()
	if p.pos < len(p.s) {
		return decimal.Zero, fmt.Errorf("trailing")
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

func (p *arithParser) parseExpr() (decimal.Decimal, error) {
	v, err := p.parseTerm()
	if err != nil {
		return decimal.Zero, err
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
			return decimal.Zero, err
		}
		if op == '+' {
			v = v.Add(r)
		} else {
			v = v.Sub(r)
		}
	}
}

func (p *arithParser) parseTerm() (decimal.Decimal, error) {
	v, err := p.parseFactor()
	if err != nil {
		return decimal.Zero, err
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
			return decimal.Zero, err
		}
		if op == '*' {
			v = v.Mul(r)
		} else if r.IsZero() {
			return decimal.Zero, fmt.Errorf("div0")
		} else {
			v = v.DivRound(r, money.RateDivPrecision)
		}
	}
}

func (p *arithParser) parseFactor() (decimal.Decimal, error) {
	p.skip()
	if p.pos < len(p.s) && p.s[p.pos] == '(' {
		p.pos++
		v, err := p.parseExpr()
		if err != nil {
			return decimal.Zero, err
		}
		p.skip()
		if p.pos < len(p.s) && p.s[p.pos] == ')' {
			p.pos++
		}
		return v, nil
	}
	if p.pos < len(p.s) && (p.s[p.pos] == '+' || p.s[p.pos] == '-') {
		neg := p.s[p.pos] == '-'
		p.pos++
		v, err := p.parseFactor()
		if neg {
			v = v.Neg()
		}
		return v, err
	}
	start := p.pos
	for p.pos < len(p.s) && (unicode.IsDigit(rune(p.s[p.pos])) || p.s[p.pos] == '.') {
		p.pos++
	}
	if start == p.pos {
		return decimal.Zero, fmt.Errorf("number")
	}
	return decimal.NewFromString(p.s[start:p.pos])
}

// asDecimal coerces JSON/numeric values (and money fields) to an exact decimal.
func asDecimal(v any) (decimal.Decimal, bool) {
	switch n := v.(type) {
	case money.Money:
		return n.Decimal(), true
	case *money.Money:
		if n == nil {
			return decimal.Zero, false
		}
		return n.Decimal(), true
	case decimal.Decimal:
		return n, true
	case *decimal.Decimal:
		if n == nil {
			return decimal.Zero, false
		}
		return *n, true
	case float64:
		return decimal.NewFromFloat(n), true
	case float32:
		return decimal.NewFromFloat32(n), true
	case int:
		return decimal.NewFromInt(int64(n)), true
	case int64:
		return decimal.NewFromInt(n), true
	case json.Number:
		d, err := decimal.NewFromString(n.String())
		return d, err == nil
	case string:
		d, err := decimal.NewFromString(n)
		return d, err == nil
	}
	return decimal.Zero, false
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
