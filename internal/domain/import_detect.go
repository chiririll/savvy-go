package domain

import (
	"bytes"
	"math/rand"
	"strings"
	"time"
	"unicode"

	"github.com/shopspring/decimal"
)

const detectSampleSize = 300

var importDateFormats = []string{"ISO", "DD.MM.YYYY", "DD/MM/YYYY", "MM/DD/YYYY"}

var importDateLayouts = map[string][]string{
	"ISO":        {"2006-01-02"},
	"DD.MM.YYYY": {"2.1.2006", "2.1.06"},
	"DD/MM/YYYY": {"2/1/2006", "2/1/06"},
	"MM/DD/YYYY": {"1/2/2006", "1/2/06"},
}

// detectedImport is what sniffing a CSV sample produced.
type detectedImport struct {
	delimiter    rune
	dateFormat   string
	amountFormat string
	mapping      map[string]int
}

// sniffDelimiter picks the most frequent candidate delimiter on the first line.
func sniffDelimiter(raw []byte) rune {
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))
	line := raw
	if i := bytes.IndexByte(raw, '\n'); i >= 0 {
		line = raw[:i]
	}
	best, bestN := ',', 0
	for _, d := range []rune{',', ';', '\t', '|'} {
		if n := bytes.Count(line, []byte(string(d))); n > bestN {
			best, bestN = d, n
		}
	}
	return best
}

// detectImport guesses the column mapping and date/amount formats from a
// random sample of rows.
func detectImport(headers []string, rows [][]string, delimiter rune) detectedImport {
	d := detectedImport{delimiter: delimiter, dateFormat: "ISO", amountFormat: "US", mapping: suggestMapping(headers)}
	sample := sampleRows(rows, detectSampleSize)
	if len(sample) == 0 {
		return d
	}

	dateCol, ok := d.mapping["date"]
	if !ok {
		dateCol, ok = bestColumn(headers, sample, -1, func(col int) float64 { return bestDateScore(columnValues(sample, col)) })
		if ok {
			d.mapping["date"] = dateCol
		}
	}
	if ok {
		d.dateFormat = detectDateFormat(columnValues(sample, dateCol))
	}

	amountCol, ok := d.mapping["amount"]
	if !ok {
		skip := -1
		if dc, has := d.mapping["date"]; has {
			skip = dc
		}
		amountCol, ok = bestColumn(headers, sample, skip, func(col int) float64 { return numericScore(columnValues(sample, col)) })
		if ok {
			d.mapping["amount"] = amountCol
		}
	}
	if ok {
		d.amountFormat = detectAmountFormat(columnValues(sample, amountCol), delimiter)
	}
	return d
}

func sampleRows(rows [][]string, n int) [][]string {
	if len(rows) <= n {
		return rows
	}
	out := make([][]string, 0, n)
	for _, i := range rand.Perm(len(rows))[:n] {
		out = append(out, rows[i])
	}
	return out
}

func columnValues(rows [][]string, col int) []string {
	var out []string
	for _, r := range rows {
		if col < len(r) {
			if v := strings.TrimSpace(r[col]); v != "" {
				out = append(out, v)
			}
		}
	}
	return out
}

// bestColumn returns the column with the highest score, if it is convincing.
func bestColumn(headers []string, sample [][]string, skip int, score func(col int) float64) (int, bool) {
	width := len(headers)
	for _, r := range sample {
		if len(r) > width {
			width = len(r)
		}
	}
	best, bestScore := -1, 0.0
	for col := 0; col < width; col++ {
		if col == skip {
			continue
		}
		if s := score(col); s > bestScore {
			best, bestScore = col, s
		}
	}
	return best, best >= 0 && bestScore >= 0.8
}

func bestDateScore(values []string) float64 {
	best := 0.0
	for _, f := range importDateFormats {
		if s := dateScore(values, f); s > best {
			best = s
		}
	}
	return best
}

func dateScore(values []string, format string) float64 {
	if len(values) == 0 {
		return 0
	}
	ok := 0
	for _, v := range values {
		if parseImportDate(v, format) != "" {
			ok++
		}
	}
	return float64(ok) / float64(len(values))
}

// detectDateFormat returns the format that parses the most sampled values.
// Ambiguous values (01/02/2024) parse under both slash formats, but impossible
// ones (13/02/2024) only under one, so any unambiguous value decides; ties fall
// back to the order of importDateFormats.
func detectDateFormat(values []string) string {
	best, bestScore := "ISO", -1.0
	for _, f := range importDateFormats {
		if s := dateScore(values, f); s > bestScore {
			best, bestScore = f, s
		}
	}
	return best
}

func numericScore(values []string) float64 {
	if len(values) == 0 {
		return 0
	}
	ok := 0
	for _, v := range values {
		c := cleanAmount(v)
		if strings.ContainsAny(c, "0123456789") && bestDateScore([]string{v}) == 0 {
			ok++
		}
	}
	return float64(ok) / float64(len(values))
}

// detectAmountFormat decides between US (1,234.56) and EU (1.234,56).
func detectAmountFormat(values []string, delimiter rune) string {
	us, eu := 0, 0
	for _, v := range values {
		c := cleanAmount(v)
		comma, dot := strings.Count(c, ","), strings.Count(c, ".")
		switch {
		case comma > 0 && dot > 0:
			if strings.LastIndex(c, ".") > strings.LastIndex(c, ",") {
				us++
			} else {
				eu++
			}
		case comma > 1:
			us++
		case dot > 1:
			eu++
		case comma == 1:
			if len(c)-strings.Index(c, ",")-1 != 3 { // 1,5 and 1,50 are decimals; 1,234 is ambiguous
				eu++
			}
		case dot == 1:
			if len(c)-strings.Index(c, ".")-1 != 3 { // 1.5 and 1.50 are decimals; 1.234 is ambiguous
				us++
			}
		}
	}
	switch {
	case eu > us:
		return "EU"
	case us > eu:
		return "US"
	case delimiter == ';':
		return "EU"
	}
	return "US"
}

// cleanAmount keeps only digits, separators and signs; accounting-style
// negatives "(12.50)" and trailing minus "12.50-" become a leading minus.
func cleanAmount(v string) string {
	v = strings.TrimSpace(v)
	neg := strings.HasPrefix(v, "(") && strings.HasSuffix(v, ")")
	var b strings.Builder
	for _, r := range v {
		if unicode.IsDigit(r) || r == ',' || r == '.' || r == '-' || r == '+' {
			b.WriteRune(r)
		}
	}
	c := b.String()
	if strings.HasSuffix(c, "-") {
		c = "-" + strings.TrimSuffix(c, "-")
	}
	if neg && !strings.HasPrefix(c, "-") {
		c = "-" + c
	}
	return c
}

func parseImportDate(v, format string) string {
	v = strings.TrimSpace(v)
	if i := strings.IndexAny(v, "T "); i > 0 {
		v = v[:i]
	}
	layouts, ok := importDateLayouts[format]
	if !ok {
		layouts = importDateLayouts["ISO"]
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return ""
}

func parseImportAmount(v, format string) (decimal.Decimal, error) {
	c := cleanAmount(v)
	if format == "EU" {
		c = strings.ReplaceAll(c, ".", "")
		c = strings.ReplaceAll(c, ",", ".")
	} else {
		c = strings.ReplaceAll(c, ",", "")
	}
	return decimal.NewFromString(c)
}

// parseImportType maps a type cell to income/expense, or "" when unknown.
func parseImportType(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "income", "credit", "in", "+", "доход", "приход", "поступление":
		return "income"
	case "expense", "debit", "out", "-", "расход", "списание":
		return "expense"
	}
	return ""
}
