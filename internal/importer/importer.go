package importer

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	repo "savvy-go/internal/db/repo/importer"
	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
	"savvy-go/internal/store"
	"savvy-go/internal/uploads"
)

// Service imports transactions from uploaded CSV files.
type Service struct {
	DB      store.DB
	Uploads uploads.Service
	Txs     ledger.Transactions
}

func (s Service) repo() repo.Repo { return repo.Repo{DB: s.DB} }

// CountsByStatus counts the imports of the space by their status.
func (s Service) CountsByStatus(ctx context.Context) (map[string]int, error) {
	return s.repo().CountsByStatus(ctx)
}

type Import struct {
	ID            string
	UserID        *int64
	UploadID      *string
	Status        string
	Mapping       map[string]any
	Options       map[string]any
	TotalRows     *int
	ProcessedRows int
	CreatedCount  int
	SkippedCount  int
	ErrorCount    int
	Errors        []RowError
	Meta          Meta
	Message       *string
}

// RowError is a row the import could not take, numbered from 1.
type RowError struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// Formats are the CSV conventions found in the sample.
type Formats struct {
	DateFormat   string `json:"date_format"`
	AmountFormat string `json:"amount_format"`
	HasHeader    bool   `json:"has_header"`
	Delimiter    string `json:"delimiter"`
}

// ParseMeta describes how the file was read.
type ParseMeta struct {
	Delimiter string `json:"delimiter"`
	HasHeader bool   `json:"has_header"`
	Encoding  string `json:"encoding"`
}

// Meta is what an import keeps next to its row counts, stored as JSON with
// snake_case keys. The first group is set when the file is parsed, the second
// when the import completes.
type Meta struct {
	Headers          []string       `json:"headers,omitempty"`
	PreviewRows      [][]string     `json:"preview_rows,omitempty"`
	DetectedFormats  Formats        `json:"detected_formats"`
	SuggestedMapping map[string]int `json:"suggested_mapping,omitempty"`
	ParseMeta        ParseMeta      `json:"parse_meta"`

	CreatedCurrencies []string `json:"created_currencies"`
	CreatedTags       []string `json:"created_tags"`
	CreatedCategories []string `json:"created_categories"`
}

func (s Service) Create(ctx context.Context, userID int64, uploadID string) (*Import, error) {
	id := newID()
	if err := s.repo().Insert(ctx, id, userID, uploadID, "parsing", time.Now().UTC()); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Service) ByID(ctx context.Context, id string) (*Import, error) {
	row, err := s.repo().ByID(ctx, id)
	if row == nil || err != nil {
		return nil, err
	}
	im := Import{
		ID: row.ID, UserID: row.UserID, UploadID: row.UploadID, Status: row.Status, TotalRows: row.TotalRows,
		ProcessedRows: row.ProcessedRows, CreatedCount: row.CreatedCount, SkippedCount: row.SkippedCount,
		ErrorCount: row.ErrorCount, Message: row.Message,
	}
	if row.Mapping != nil {
		_ = json.Unmarshal([]byte(*row.Mapping), &im.Mapping)
	}
	if row.Options != nil {
		_ = json.Unmarshal([]byte(*row.Options), &im.Options)
	}
	if row.Errors != nil && *row.Errors != "" {
		_ = json.Unmarshal([]byte(*row.Errors), &im.Errors)
	}
	if row.Meta != nil && *row.Meta != "" {
		_ = json.Unmarshal([]byte(*row.Meta), &im.Meta)
	}
	return &im, nil
}

func (s Service) Parse(ctx context.Context, importID, uploadID string) error {
	im, err := s.ByID(ctx, importID)
	if err != nil || im == nil {
		return err
	}
	up, err := s.Uploads.ByID(ctx, uploadID)
	if err != nil || up == nil || !up.Completed() {
		return s.fail(ctx, importID, "The uploaded file is unavailable.")
	}
	raw, err := s.Uploads.ReadFile(up)
	if err != nil {
		return s.fail(ctx, importID, err.Error())
	}
	delimiter, headers, rows, err := parseCSVWithDelimiter(raw)
	if err != nil {
		return s.fail(ctx, importID, err.Error())
	}
	detected := detectImport(headers, rows, delimiter)
	preview := rows
	if len(preview) > 10 {
		preview = preview[:10]
	}
	delim := string(detected.delimiter)
	meta := Meta{
		Headers: headers, PreviewRows: preview,
		DetectedFormats: Formats{
			DateFormat:   detected.dateFormat,
			AmountFormat: detected.amountFormat,
			HasHeader:    true,
			Delimiter:    delim,
		},
		SuggestedMapping: detected.mapping,
		ParseMeta:        ParseMeta{Delimiter: delim, HasHeader: true, Encoding: "utf-8"},
	}
	rawMeta, _ := json.Marshal(meta)
	return s.repo().MarkParsed(ctx, importID, len(rows), string(rawMeta), time.Now().UTC())
}

func (s Service) Execute(ctx context.Context, importID string, mapping, options map[string]any) error {
	im, err := s.ByID(ctx, importID)
	if err != nil || im == nil {
		return err
	}
	if im.UploadID == nil {
		return s.fail(ctx, importID, "The uploaded file is gone.")
	}
	up, err := s.Uploads.ByID(ctx, *im.UploadID)
	if err != nil || up == nil || !up.Completed() {
		return s.fail(ctx, importID, "The uploaded file is gone.")
	}
	mapJSON, _ := json.Marshal(mapping)
	optJSON, _ := json.Marshal(options)
	now := time.Now().UTC()
	_ = s.repo().MarkImporting(ctx, importID, string(mapJSON), string(optJSON), now)

	raw, err := s.Uploads.ReadFile(up)
	if err != nil {
		return s.fail(ctx, importID, err.Error())
	}
	_, rows, err := parseCSV(raw)
	if err != nil {
		return s.fail(ctx, importID, err.Error())
	}
	created, skipped := 0, 0
	errs := []RowError{}
	accountID, _ := asInt64(options["default_account_id"])
	resolver, err := s.newCategoryResolver(ctx, collectImportCategories(rows, mapping, options), options)
	if err != nil {
		return s.fail(ctx, importID, err.Error())
	}
	unit, err := ledger.Accounts{DB: s.DB}.Unit(ctx, accountID)
	if err != nil {
		return s.fail(ctx, importID, err.Error())
	}
	for i, row := range rows {
		res := processImportRow(row, mapping, options, i+1)
		if res.err != "" {
			if len(errs) < 200 {
				errs = append(errs, RowError{Row: i + 1, Message: res.err})
			}
			continue
		}
		amount, err := money.FromInput(res.amount, unit)
		if err != nil {
			if len(errs) < 200 {
				errs = append(errs, RowError{Row: i + 1, Message: err.Error()})
			}
			continue
		}
		hash := dedupHash(res.date, res.amount, res.desc)
		added, err := s.repo().InsertRow(ctx, repo.Row{
			Type: res.typ, AccountID: accountID, CategoryID: resolver.resolve(res.category), Minor: amount.Minor(),
			Description: res.desc, Date: res.date, Status: "confirmed", DedupHash: hash,
		}, now)
		if err != nil {
			errs = append(errs, RowError{Row: i + 1, Message: err.Error()})
			continue
		}
		if added {
			created++
		} else {
			skipped++
		}
	}
	errJSON, _ := json.Marshal(errs)
	meta := im.Meta
	meta.CreatedCurrencies, meta.CreatedTags, meta.CreatedCategories = []string{}, []string{}, resolver.created
	metaJSON, _ := json.Marshal(meta)
	err = s.repo().MarkCompleted(ctx, importID, repo.Result{
		Processed: created + skipped + len(errs), Created: created, Skipped: skipped, Errors: len(errs),
		ErrorsJSON: string(errJSON), MetaJSON: string(metaJSON),
	}, now)
	if err != nil {
		return err
	}
	_ = s.Uploads.Discard(ctx, up)
	return nil
}

func (s Service) Preview(ctx context.Context, im *Import, mapping, options map[string]any) (*PreviewResult, error) {
	if im.UploadID == nil {
		return nil, fmt.Errorf("file gone")
	}
	up, err := s.Uploads.ByID(ctx, *im.UploadID)
	if err != nil || up == nil || !up.Completed() {
		return nil, fmt.Errorf("file gone")
	}
	raw, err := s.Uploads.ReadFile(up)
	if err != nil {
		return nil, err
	}
	_, rows, err := parseCSV(raw)
	if err != nil {
		return nil, err
	}
	cats := collectImportCategories(rows, mapping, options)
	if err := s.matchImportCategories(ctx, cats); err != nil {
		return nil, err
	}
	var preview []PreviewRow
	willCreate, willSkip, hasErrors := 0, 0, 0
	for i, row := range rows {
		res := processImportRow(row, mapping, options, i+1)
		status := "new"
		if res.err != "" {
			hasErrors++
			status = "error"
		} else {
			willCreate++
		}
		if len(preview) < 200 {
			preview = append(preview, PreviewRow{
				Row: i + 1, Date: res.date, Type: res.typ, Amount: res.amount,
				Description: res.desc, Status: status, Error: nilOr(res.err), Category: nilOr(res.category),
			})
		}
	}
	_ = willSkip
	categories, categoriesToCreate := []PreviewCategory{}, []string{}
	for _, c := range cats {
		if c.MatchID == nil {
			categoriesToCreate = append(categoriesToCreate, c.Name)
		}
		categories = append(categories, PreviewCategory{Name: c.Name, Type: c.Type, Count: c.Count, MatchID: c.MatchID})
	}
	return &PreviewResult{
		Transactions: preview,
		Summary: PreviewSummary{
			WillCreate: willCreate, WillSkip: willSkip, HasErrors: hasErrors,
			TotalRows: im.TotalRows, Sampled: willCreate + willSkip + hasErrors,
			CategoriesToCreate: categoriesToCreate, Categories: categories,
		},
	}, nil
}

func (s Service) fail(ctx context.Context, id, msg string) error {
	return s.repo().Fail(ctx, id, msg, time.Now().UTC())
}

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

func processImportRow(row []string, mapping, options map[string]any, _ int) importRow {
	out := importRow{typ: "expense"}
	if v, _ := options["default_type"].(string); v != "" {
		out.typ = v
	}
	di, ok := mappingIndex(mapping, "date")
	if !ok {
		out.err = "Date column is not mapped."
		return out
	}
	if di >= len(row) {
		out.err = "Row is too short."
		return out
	}
	out.date = parseImportDate(row[di], optString(options, "date_format"))
	if out.date == "" {
		out.err = "Invalid date: " + row[di]
		return out
	}
	if ledger.IsFuture(out.date) {
		out.err = "Date is in the future."
		return out
	}
	ai, ok := mappingIndex(mapping, "amount")
	if !ok {
		out.err = "Amount column is not mapped."
		return out
	}
	if ai >= len(row) {
		out.err = "Row is too short."
		return out
	}
	amt, err := parseImportAmount(row[ai], optString(options, "amount_format"))
	if err != nil {
		out.err = "Invalid amount: " + row[ai]
		return out
	}
	out.amount = amt
	typeKnown := false
	if ti, ok := mappingIndex(mapping, "type"); ok && ti < len(row) {
		if t := parseImportType(row[ti]); t != "" {
			out.typ, typeKnown = t, true
		}
	}
	if !typeKnown {
		if amt.Sign() < 0 {
			out.typ = "expense"
		} else if amt.Sign() > 0 {
			out.typ = "income"
		}
	}
	out.amount = out.amount.Abs()
	if di, ok := mappingIndex(mapping, "description"); ok && di < len(row) {
		out.desc = strings.TrimSpace(row[di])
	}
	if ci, ok := mappingIndex(mapping, "category"); ok && ci < len(row) {
		out.category = strings.TrimSpace(row[ci])
	}
	return out
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

func nilOr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
