package importer

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
)

func parseCSV(raw []byte) ([]string, [][]string, error) {
	_, headers, rows, err := parseCSVWithDelimiter(raw)
	return headers, rows, err
}

func parseCSVWithDelimiter(raw []byte) (rune, []string, [][]string, error) {
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))
	delimiter := sniffDelimiter(raw)
	r := csv.NewReader(bytes.NewReader(raw))
	r.Comma = delimiter
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	all, err := r.ReadAll()
	if err != nil {
		return 0, nil, nil, err
	}
	if len(all) == 0 {
		return 0, nil, nil, fmt.Errorf("empty csv")
	}
	return delimiter, all[0], all[1:], nil
}

func suggestMapping(headers []string) map[string]int {
	out := map[string]int{}
	for i, h := range headers {
		key := ""
		switch strings.ToLower(strings.TrimSpace(h)) {
		case "date", "дата":
			key = "date"
		case "amount", "sum", "сумма":
			key = "amount"
		case "description", "memo", "details", "описание", "назначение":
			key = "description"
		case "type", "тип":
			key = "type"
		case "category", "категория":
			key = "category"
		case "tags", "tag", "теги", "тег":
			key = "tags"
		case "currency", "валюта":
			key = "currency"
		}
		if _, taken := out[key]; key != "" && !taken {
			out[key] = i
		}
	}
	return out
}

type importRow struct {
	date, typ, desc, category, err string
	amount                         decimal.Decimal
}

// cellAt is the cell of the column mapped to key. It is "" and false where the
// column is not mapped or the row is too short for it.
func cellAt(row []string, mapping map[string]any, key string) (string, bool) {
	i, ok := mappingIndex(mapping, key)
	if !ok || i >= len(row) {
		return "", false
	}
	return row[i], true
}

// requiredCell is cellAt for a column the import cannot do without; msg says
// what is wrong when it is not there.
func requiredCell(row []string, mapping map[string]any, key, label string) (cell, msg string) {
	if _, ok := mappingIndex(mapping, key); !ok {
		return "", label + " column is not mapped."
	}
	cell, ok := cellAt(row, mapping, key)
	if !ok {
		return "", "Row is too short."
	}
	return cell, ""
}

func processImportRow(row []string, mapping, options map[string]any, _ int) importRow {
	out := importRow{typ: "expense"}
	if v, _ := options["default_type"].(string); v != "" {
		out.typ = v
	}
	dateCell, msg := requiredCell(row, mapping, "date", "Date")
	if msg != "" {
		out.err = msg
		return out
	}
	out.date = parseImportDate(dateCell, optString(options, "date_format"))
	if out.date == "" {
		out.err = "Invalid date: " + dateCell
		return out
	}
	if ledger.IsFuture(out.date) {
		out.err = "Date is in the future."
		return out
	}
	amountCell, msg := requiredCell(row, mapping, "amount", "Amount")
	if msg != "" {
		out.err = msg
		return out
	}
	amt, err := parseImportAmount(amountCell, optString(options, "amount_format"))
	if err != nil {
		out.err = "Invalid amount: " + amountCell
		return out
	}
	out.typ = importType(row, mapping, amt, out.typ)
	out.amount = amt.Abs()
	if v, ok := cellAt(row, mapping, "description"); ok {
		out.desc = strings.TrimSpace(v)
	}
	if v, ok := cellAt(row, mapping, "category"); ok {
		out.category = strings.TrimSpace(v)
	}
	return out
}

// importType is the type column's value when it names one, else the sign of
// the amount, else fallback.
func importType(row []string, mapping map[string]any, amt decimal.Decimal, fallback string) string {
	if v, ok := cellAt(row, mapping, "type"); ok {
		if t := parseImportType(v); t != "" {
			return t
		}
	}
	switch {
	case amt.Sign() < 0:
		return "expense"
	case amt.Sign() > 0:
		return "income"
	}
	return fallback
}

func optString(options map[string]any, key string) string {
	v, _ := options[key].(string)
	return v
}

func mappingIndex(mapping map[string]any, key string) (int, bool) {
	v, ok := mapping[key]
	if !ok {
		return 0, false
	}
	n, ok := asInt64(v)
	return int(n), ok
}

// dedupHash identifies a row by its date, amount and description. The amount keeps
// every decimal it has, at least two: cutting it to cents would make 0.0000015 and
// 0.0000025 of an eight-decimal currency the same row.
func dedupHash(date string, amount decimal.Decimal, desc string) string {
	norm := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(desc)), " "))
	places := 2
	if _, frac, ok := strings.Cut(amount.String(), "."); ok { // String drops trailing zeros
		places = max(places, len(frac))
	}
	sum := md5.Sum([]byte(fmt.Sprintf("%s|%s|%s", date, amount.StringFixed(int32(places)), norm)))
	return hex.EncodeToString(sum[:])
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// asInt64 reads a whole number from decoded JSON, a number or a numeric string.
func asInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(math.Round(n)), true
	case int:
		return int64(n), true
	case int64:
		return n, true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	case string:
		i, err := strconv.ParseInt(n, 10, 64)
		return i, err == nil
	}
	return 0, false
}

func ptrIfSet(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
