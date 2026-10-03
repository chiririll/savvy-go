package domain

import (
	"bytes"
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/money"
)

type Imports struct {
	DB      *sql.DB
	Uploads Uploads
	Txs     Transactions
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
	Errors        any
	Meta          map[string]any
	Message       *string
}

func (s Imports) Create(ctx context.Context, userID int64, uploadID string) (*Import, error) {
	id := newUploadID()
	now := time.Now().UTC().Format(time.RFC3339)
	err := db.Q(s.DB).InsertImport(ctx, sqlc.InsertImportParams{
		ID: id, UserID: db.NI(userID), UploadID: db.NS(uploadID), Status: "parsing",
		CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
	})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Imports) ByID(ctx context.Context, id string) (*Import, error) {
	row, err := db.Q(s.DB).GetImport(ctx, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	im := Import{
		ID: row.ID, Status: row.Status, ProcessedRows: int(row.ProcessedRows),
		CreatedCount: int(row.CreatedCount), SkippedCount: int(row.SkippedCount), ErrorCount: int(row.ErrorCount),
	}
	if row.UserID.Valid {
		im.UserID = &row.UserID.Int64
	}
	if row.UploadID.Valid {
		im.UploadID = &row.UploadID.String
	}
	if row.TotalRows.Valid {
		n := int(row.TotalRows.Int64)
		im.TotalRows = &n
	}
	if row.Mapping.Valid {
		_ = json.Unmarshal([]byte(row.Mapping.String), &im.Mapping)
	}
	if row.Options.Valid {
		_ = json.Unmarshal([]byte(row.Options.String), &im.Options)
	}
	if row.Errors.Valid && row.Errors.String != "" {
		var v any
		_ = json.Unmarshal([]byte(row.Errors.String), &v)
		im.Errors = v
	}
	if row.Meta.Valid && row.Meta.String != "" {
		_ = json.Unmarshal([]byte(row.Meta.String), &im.Meta)
	}
	if im.Meta == nil {
		im.Meta = map[string]any{}
	}
	if row.Message.Valid {
		im.Message = &row.Message.String
	}
	return &im, nil
}

func (s Imports) Parse(ctx context.Context, importID, uploadID string) error {
	im, err := s.ByID(ctx, importID)
	if err != nil || im == nil {
		return err
	}
	up, err := s.Uploads.ByID(ctx, uploadID)
	if err != nil || up == nil || up.Status != uploadCompleted {
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
	meta := map[string]any{
		"headers": headers, "preview_rows": preview,
		"detected_formats": map[string]any{
			"date_format": detected.dateFormat, "amount_format": detected.amountFormat,
			"has_header": true, "delimiter": string(detected.delimiter),
		},
		"suggested_mapping": detected.mapping,
		"parse_meta":        map[string]any{"delimiter": string(detected.delimiter), "has_header": true, "encoding": "utf-8"},
	}
	rawMeta, _ := json.Marshal(meta)
	now := time.Now().UTC().Format(time.RFC3339)
	return db.Q(s.DB).MarkImportParsed(ctx, sqlc.MarkImportParsedParams{
		TotalRows: db.NI(int64(len(rows))), Meta: db.NS(string(rawMeta)), UpdatedAt: db.NS(now), ID: importID,
	})
}

func (s Imports) Execute(ctx context.Context, importID string, mapping, options map[string]any) error {
	im, err := s.ByID(ctx, importID)
	if err != nil || im == nil {
		return err
	}
	if im.UploadID == nil {
		return s.fail(ctx, importID, "The uploaded file is gone.")
	}
	up, err := s.Uploads.ByID(ctx, *im.UploadID)
	if err != nil || up == nil || up.Status != uploadCompleted {
		return s.fail(ctx, importID, "The uploaded file is gone.")
	}
	mapJSON, _ := json.Marshal(mapping)
	optJSON, _ := json.Marshal(options)
	now := time.Now().UTC().Format(time.RFC3339)
	_ = db.Q(s.DB).MarkImportImporting(ctx, sqlc.MarkImportImportingParams{
		Mapping: db.NS(string(mapJSON)), Options: db.NS(string(optJSON)), UpdatedAt: db.NS(now), ID: importID,
	})

	raw, err := s.Uploads.ReadFile(up)
	if err != nil {
		return s.fail(ctx, importID, err.Error())
	}
	_, rows, err := parseCSV(raw)
	if err != nil {
		return s.fail(ctx, importID, err.Error())
	}
	created, skipped := 0, 0
	var errs []map[string]any
	accountID, _ := asInt64(options["default_account_id"])
	resolver, err := s.newCategoryResolver(ctx, collectImportCategories(rows, mapping, options), options)
	if err != nil {
		return s.fail(ctx, importID, err.Error())
	}
	dec := Accounts{DB: s.DB}.Decimals(ctx, accountID)
	for i, row := range rows {
		res := processImportRow(row, mapping, options, i+1)
		if res.err != "" {
			if len(errs) < 200 {
				errs = append(errs, map[string]any{"row": i + 1, "message": res.err})
			}
			continue
		}
		hash := dedupHash(res.date, res.amount, res.desc)
		st := "confirmed"
		ins, err := db.Q(s.DB).InsertTransactionIgnoreDup(ctx, sqlc.InsertTransactionIgnoreDupParams{
			Type: res.typ, AccountID: accountID, CategoryID: resolver.resolve(res.category), Amount: money.ToMinor(res.amount, dec), Description: db.NS(res.desc),
			Date: db.NS(res.date), Status: st, DedupHash: db.NS(hash), CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
		})
		if err != nil {
			errs = append(errs, map[string]any{"row": i + 1, "message": err.Error()})
			continue
		}
		n, _ := ins.RowsAffected()
		if n == 0 {
			skipped++
		} else {
			created++
		}
	}
	errJSON, _ := json.Marshal(errs)
	meta := im.Meta
	if meta == nil {
		meta = map[string]any{}
	}
	meta["created_currencies"] = []any{}
	meta["created_tags"] = []any{}
	meta["created_categories"] = resolver.created
	metaJSON, _ := json.Marshal(meta)
	err = db.Q(s.DB).MarkImportCompleted(ctx, sqlc.MarkImportCompletedParams{
		ProcessedRows: int64(created + skipped + len(errs)), CreatedCount: int64(created), SkippedCount: int64(skipped),
		ErrorCount: int64(len(errs)), Errors: db.NS(string(errJSON)), Meta: db.NS(string(metaJSON)),
		UpdatedAt: db.NS(now), ID: importID,
	})
	if err != nil {
		return err
	}
	_ = s.Uploads.Discard(ctx, up)
	return nil
}

func (s Imports) Preview(ctx context.Context, im *Import, mapping, options map[string]any) (map[string]any, error) {
	if im.UploadID == nil {
		return nil, fmt.Errorf("file gone")
	}
	up, err := s.Uploads.ByID(ctx, *im.UploadID)
	if err != nil || up == nil || up.Status != uploadCompleted {
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
	var preview []map[string]any
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
			preview = append(preview, map[string]any{
				"row": i + 1, "date": res.date, "type": res.typ, "amount": money.Plain(res.amount),
				"description": res.desc, "status": status, "error": nilOr(res.err),
				"category": nilOr(res.category), "tags": []string{}, "duplicate_of": nil, "warnings": []string{},
			})
		}
	}
	_ = willSkip
	categories, categoriesToCreate := []map[string]any{}, []string{}
	for _, c := range cats {
		var matchID any
		if c.MatchID != nil {
			matchID = *c.MatchID
		} else {
			categoriesToCreate = append(categoriesToCreate, c.Name)
		}
		categories = append(categories, map[string]any{"name": c.Name, "type": c.Type, "count": c.Count, "match_id": matchID})
	}
	return map[string]any{
		"preview_transactions": preview,
		"summary": map[string]any{
			"will_create": willCreate, "will_skip": willSkip, "has_errors": hasErrors,
			"total_rows": im.TotalRows, "sampled": willCreate + willSkip + hasErrors,
			"currencies_to_create": []any{}, "tags_to_create": []any{},
			"categories_to_create": categoriesToCreate, "categories": categories,
		},
	}, nil
}

func (s Imports) fail(ctx context.Context, id, msg string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return db.Q(s.DB).FailImport(ctx, sqlc.FailImportParams{Message: db.NS(msg), UpdatedAt: db.NS(now), ID: id})
}

func parseCSV(raw []byte) ([]string, [][]string, error) {
	_, headers, rows, err := parseCSVWithDelimiter(raw)
	return headers, rows, err
}

func parseCSVWithDelimiter(raw []byte) (rune, []string, [][]string, error) {
	raw = bytes.TrimPrefix(raw, []byte("ï»¿"))
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
	if !ok || di >= len(row) {
		out.err = "Date column is not mapped."
		return out
	}
	out.date = parseImportDate(row[di], optString(options, "date_format"))
	if out.date == "" {
		out.err = "Invalid date: " + row[di]
		return out
	}
	if isFuture(out.date) {
		out.err = "Date is in the future."
		return out
	}
	ai, ok := mappingIndex(mapping, "amount")
	if !ok || ai >= len(row) {
		out.err = "Amount column is not mapped."
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

func dedupHash(date string, amount decimal.Decimal, desc string) string {
	norm := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(desc)), " "))
	sum := md5.Sum([]byte(fmt.Sprintf("%s|%s|%s", date, amount.StringFixed(2), norm)))
	return hex.EncodeToString(sum[:])
}
