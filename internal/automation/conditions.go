package automation

import (
	"regexp"
	"strings"

	"savvy-go/internal/ledger"
)

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
