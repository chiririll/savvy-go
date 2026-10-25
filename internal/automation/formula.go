package automation

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
)

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
