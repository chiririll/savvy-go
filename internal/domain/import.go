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
	"strconv"
	"strings"
	"time"
)

type Imports struct {
	DB      *sql.DB
	Uploads Uploads
	Txs     Transactions
}

type Import struct {
	ID             string
	UserID         *int64
	UploadID       *string
	Status         string
	Mapping        map[string]any
	Options        map[string]any
	TotalRows      *int
	ProcessedRows  int
	CreatedCount   int
	SkippedCount   int
	ErrorCount     int
	Errors         any
	Meta           map[string]any
	Message        *string
}

func (s Imports) Create(ctx context.Context, userID int64, uploadID string) (*Import, error) {
	id := newUploadID()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO transaction_imports (id, user_id, upload_id, status, created_at, updated_at)
		VALUES (?,?,?,?,?,?)`, id, userID, uploadID, "parsing", now, now)
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Imports) ByID(ctx context.Context, id string) (*Import, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, user_id, upload_id, status, mapping, options, total_rows, processed_rows,
			created_count, skipped_count, error_count, errors, meta, message
		FROM transaction_imports WHERE id = ?`, id)
	var im Import
	var user sql.NullInt64
	var upload, mapping, options, errors, meta, msg sql.NullString
	var total sql.NullInt64
	err := row.Scan(&im.ID, &user, &upload, &im.Status, &mapping, &options, &total, &im.ProcessedRows,
		&im.CreatedCount, &im.SkippedCount, &im.ErrorCount, &errors, &meta, &msg)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if user.Valid {
		im.UserID = &user.Int64
	}
	if upload.Valid {
		im.UploadID = &upload.String
	}
	if total.Valid {
		n := int(total.Int64)
		im.TotalRows = &n
	}
	if mapping.Valid {
		_ = json.Unmarshal([]byte(mapping.String), &im.Mapping)
	}
	if options.Valid {
		_ = json.Unmarshal([]byte(options.String), &im.Options)
	}
	if errors.Valid && errors.String != "" {
		var v any
		_ = json.Unmarshal([]byte(errors.String), &v)
		im.Errors = v
	}
	if meta.Valid && meta.String != "" {
		_ = json.Unmarshal([]byte(meta.String), &im.Meta)
	}
	if im.Meta == nil {
		im.Meta = map[string]any{}
	}
	if msg.Valid {
		im.Message = &msg.String
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
	headers, rows, err := parseCSV(raw)
	if err != nil {
		return s.fail(ctx, importID, err.Error())
	}
	preview := rows
	if len(preview) > 10 {
		preview = preview[:10]
	}
	meta := map[string]any{
		"headers": headers, "preview_rows": preview,
		"detected_formats":  map[string]any{"date": "ISO", "amount": "US"},
		"suggested_mapping": suggestMapping(headers),
		"parse_meta":        map[string]any{"delimiter": ",", "has_header": true, "encoding": "utf-8"},
	}
	rawMeta, _ := json.Marshal(meta)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.DB.ExecContext(ctx, `
		UPDATE transaction_imports SET status='parsed', total_rows=?, meta=?, updated_at=? WHERE id=?`,
		len(rows), string(rawMeta), now, importID)
	return err
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
	_, _ = s.DB.ExecContext(ctx, `
		UPDATE transaction_imports SET status='importing', mapping=?, options=?, processed_rows=0,
			created_count=0, skipped_count=0, error_count=0, errors=NULL, message=NULL, updated_at=? WHERE id=?`,
		string(mapJSON), string(optJSON), now, importID)

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
		ins, err := s.DB.ExecContext(ctx, `
			INSERT OR IGNORE INTO transactions (type, account_id, category_id, amount, description, date, status, dedup_hash, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?)`,
			res.typ, accountID, nil, res.amount, res.desc, res.date, st, hash, now, now)
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
	meta["created_categories"] = []any{}
	metaJSON, _ := json.Marshal(meta)
	_, err = s.DB.ExecContext(ctx, `
		UPDATE transaction_imports SET status='completed', processed_rows=?, created_count=?, skipped_count=?,
			error_count=?, errors=?, meta=?, updated_at=? WHERE id=?`,
		created+skipped+len(errs), created, skipped, len(errs), string(errJSON), string(metaJSON), now, importID)
	if err != nil {
		return err
	}
	_ = s.Uploads.Discard(ctx, up)
	return nil
}

func (im Import) JSON() map[string]any {
	parsed := im.Status == "parsed" || im.Status == "importing" || im.Status == "completed"
	m := map[string]any{
		"import_id": im.ID, "status": im.Status, "total_rows": im.TotalRows,
		"processed_rows": im.ProcessedRows, "created": im.CreatedCount,
		"skipped": im.SkippedCount, "errors": im.ErrorCount, "message": im.Message,
	}
	if parsed && im.Meta != nil {
		m["parse"] = map[string]any{
			"headers": im.Meta["headers"], "preview_rows": im.Meta["preview_rows"],
			"total_rows": im.TotalRows, "detected_formats": im.Meta["detected_formats"],
			"suggested_mapping": im.Meta["suggested_mapping"],
		}
	} else {
		m["parse"] = nil
	}
	if im.Status == "completed" {
		m["result"] = map[string]any{
			"created": im.CreatedCount, "skipped_duplicates": im.SkippedCount,
			"errors": im.Errors, "created_currencies": im.Meta["created_currencies"],
			"created_tags": im.Meta["created_tags"], "created_categories": im.Meta["created_categories"],
		}
	} else {
		m["result"] = nil
	}
	return m
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
				"row": i + 1, "date": res.date, "type": res.typ, "amount": res.amount,
				"description": res.desc, "status": status, "error": nilOr(res.err),
			})
		}
	}
	_ = willSkip
	return map[string]any{
		"preview_transactions": preview,
		"summary": map[string]any{
			"will_create": willCreate, "will_skip": willSkip, "has_errors": hasErrors,
			"total_rows": im.TotalRows, "sampled": willCreate + willSkip + hasErrors,
			"currencies_to_create": []any{}, "tags_to_create": []any{}, "categories_to_create": []any{},
		},
	}, nil
}

func (s Imports) fail(ctx context.Context, id, msg string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.ExecContext(ctx, `UPDATE transaction_imports SET status='failed', message=?, updated_at=? WHERE id=?`, msg, now, id)
	return err
}

func parseCSV(raw []byte) ([]string, [][]string, error) {
	r := csv.NewReader(bytes.NewReader(raw))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	all, err := r.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	if len(all) == 0 {
		return nil, nil, fmt.Errorf("empty csv")
	}
	return all[0], all[1:], nil
}

func suggestMapping(headers []string) map[string]int {
	out := map[string]int{}
	for i, h := range headers {
		switch strings.ToLower(strings.TrimSpace(h)) {
		case "date":
			out["date"] = i
		case "amount":
			out["amount"] = i
		case "description", "memo", "details":
			out["description"] = i
		case "type":
			out["type"] = i
		case "category":
			out["category"] = i
		case "tags", "tag":
			out["tags"] = i
		}
	}
	return out
}

type importRow struct {
	date, typ, desc, err string
	amount               float64
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
	out.date = parseImportDate(row[di], fmt.Sprint(options["date_format"]))
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
	amt, err := parseImportAmount(row[ai])
	if err != nil {
		out.err = "Invalid amount: " + row[ai]
		return out
	}
	out.amount = amt
	if _, hasType := mapping["type"]; !hasType {
		if amt < 0 {
			out.typ = "expense"
		} else if amt > 0 {
			out.typ = "income"
		}
	}
	out.amount = absFloat(out.amount)
	if di, ok := mappingIndex(mapping, "description"); ok && di < len(row) {
		out.desc = strings.TrimSpace(row[di])
	}
	return out
}

func mappingIndex(mapping map[string]any, key string) (int, bool) {
	v, ok := mapping[key]
	if !ok {
		return 0, false
	}
	n, ok := asInt64(v)
	return int(n), ok
}

func parseImportDate(v, format string) string {
	v = strings.TrimSpace(v)
	for _, layout := range []string{"2006-01-02", "02/01/2006", "01/02/2006", "2 Jan 2006", time.RFC3339} {
		if t, err := time.Parse(layout, v); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return ""
}

func parseImportAmount(v string) (float64, error) {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, ",", "")
	v = strings.ReplaceAll(v, " ", "")
	return strconv.ParseFloat(v, 64)
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func dedupHash(date string, amount float64, desc string) string {
	norm := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(desc)), " "))
	sum := md5.Sum([]byte(fmt.Sprintf("%s|%.2f|%s", date, amount, norm)))
	return hex.EncodeToString(sum[:])
}
