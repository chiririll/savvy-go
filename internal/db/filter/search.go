package filter

import (
	"regexp"
	"strconv"
	"strings"

	"savvy-go/internal/db"
	"savvy-go/internal/money"
)

// A free-text search is a list of terms, all of which must match a
// transaction. A term matches when the transaction's description or one of its
// item names contains it (any case, any script), or, when the term reads as a
// number, when that number is the transaction's amount, its received amount or
// the price of one of its items.

// amountScales is the number of currency decimal places a minor-unit amount is
// scaled by; the schema allows 0 through 12.
const maxScale = 12

// amount is a number typed in a search: Minor / 10^Frac, with no trailing zero
// in the fraction, so 1879.50 is {18795, 1}.
type amount struct {
	Minor int64
	Frac  int
}

type searchTerm struct {
	Text    string // lower case
	Amounts []amount
}

// groupedNumber is a number with spaces between thousands, "1 879" or "12 345.60".
var groupedNumber = regexp.MustCompile(`\b\d{1,3}(?:[\s\x{a0}\x{202f}]\d{3})+(?:[.,]\d+)?\b`)

var spaces = regexp.MustCompile(`[\s\x{a0}\x{202f}]+`)

var plainNumber = regexp.MustCompile(`^\d+(?:[.,]\d+)*$`)

// Bounds on one search, so a request cannot build an unbounded statement.
const (
	maxSearchRunes = 200
	maxSearchTerms = 8
)

func parseSearch(q string) []searchTerm {
	if r := []rune(q); len(r) > maxSearchRunes {
		q = string(r[:maxSearchRunes])
	}
	q = groupedNumber.ReplaceAllStringFunc(q, func(s string) string { return spaces.ReplaceAllString(s, "") })
	var terms []searchTerm
	for _, w := range strings.Fields(q) {
		if len(terms) == maxSearchTerms {
			break
		}
		terms = append(terms, searchTerm{Text: strings.ToLower(w), Amounts: amountsOf(w)})
	}
	return terms
}

// amountsOf reads w as a number. "1,879" is ambiguous (1879 or 1.879, in a
// currency with three decimals), so every reading is returned.
func amountsOf(w string) []amount {
	if !plainNumber.MatchString(w) {
		return nil
	}
	lastDot, lastComma := strings.LastIndex(w, "."), strings.LastIndex(w, ",")
	var readings []string // integer part + "." + fraction
	switch {
	case lastDot < 0 && lastComma < 0:
		readings = []string{w}
	case lastDot >= 0 && lastComma >= 0:
		// Both kinds: the one that comes last is the decimal mark.
		sep, group := ".", ","
		if lastComma > lastDot {
			sep, group = ",", "."
		}
		i := strings.LastIndex(w, sep)
		readings = []string{strings.ReplaceAll(w[:i], group, "") + "." + w[i+1:]}
	default:
		sep := "."
		if lastComma >= 0 {
			sep = ","
		}
		i := strings.LastIndex(w, sep)
		whole, frac := w[:i], w[i+1:]
		if strings.Count(w, sep) > 1 {
			// "1,879,000": the mark repeats, so it only groups thousands.
			readings = []string{strings.ReplaceAll(w, sep, "")}
			break
		}
		readings = []string{whole + "." + frac}
		if len(frac) == 3 && len(whole) <= 3 {
			readings = append(readings, whole+frac)
		}
	}
	var out []amount
	for _, r := range readings {
		if a, ok := toAmount(r); ok {
			out = append(out, a)
		}
	}
	return out
}

func toAmount(s string) (amount, bool) {
	whole, frac, _ := strings.Cut(s, ".")
	frac = strings.TrimRight(frac, "0")
	if len(frac) > maxScale || len(whole)+len(frac) > 18 {
		return amount{}, false
	}
	minor, err := strconv.ParseInt(whole+frac, 10, 64)
	if err != nil || minor > money.MaxMinor {
		return amount{}, false
	}
	return amount{Minor: minor, Frac: len(frac)}, true
}

// searchWhere is the condition for a search; "" and nil args when it has no
// terms. It needs the joins of txJoins: c is the currency of the transaction's
// account and cb that of the account it goes to.
func searchWhere(q string) (string, []any) {
	var sb strings.Builder
	var args []any
	for _, term := range parseSearch(q) {
		sb.WriteString(` AND (instr(` + db.FoldFunc + `(COALESCE(t.description, '')), ?) > 0` +
			` OR EXISTS (SELECT 1 FROM transaction_items i WHERE i.transaction_id = t.id` +
			` AND instr(` + db.FoldFunc + `(i.name), ?) > 0)`)
		args = append(args, term.Text, term.Text)
		for _, a := range term.Amounts {
			// The stored integer is the number scaled by the currency's
			// decimals, which only a currency with at least as many
			// decimals as the number has can represent.
			from := `(c.decimals >= ? AND %s = ? * ` + db.Pow10Func + `(c.decimals - ?))`
			sb.WriteString(` OR ` + strings.ReplaceAll(from, "%s", "t.amount"))
			args = append(args, a.Frac, a.Minor, a.Frac)
			sb.WriteString(` OR EXISTS (SELECT 1 FROM transaction_items i WHERE i.transaction_id = t.id` +
				` AND c.decimals >= ? AND (i.total_price = ? * ` + db.Pow10Func + `(c.decimals - ?)` +
				` OR i.price_per_unit = ? * ` + db.Pow10Func + `(c.decimals - ?)))`)
			args = append(args, a.Frac, a.Minor, a.Frac, a.Minor, a.Frac)
			sb.WriteString(` OR (cb.decimals >= ? AND t.to_amount = ? * ` + db.Pow10Func + `(cb.decimals - ?))`)
			args = append(args, a.Frac, a.Minor, a.Frac)
		}
		sb.WriteString(`)`)
	}
	return sb.String(), args
}
