package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
	accountID, _ := asInt64(options["default_account_id"])
	resolver, err := s.newCategoryResolver(ctx, collectImportCategories(rows, mapping, options), options)
	if err != nil {
		return s.fail(ctx, importID, err.Error())
	}
	unit, err := ledger.Accounts{DB: s.DB}.Unit(ctx, accountID)
	if err != nil {
		return s.fail(ctx, importID, err.Error())
	}
	run := importRun{s: s, accountID: accountID, unit: unit, resolver: resolver, now: now, errs: []RowError{}}
	for i, row := range rows {
		run.row(ctx, row, mapping, options, i+1)
	}
	errJSON, _ := json.Marshal(run.errs)
	meta := im.Meta
	meta.CreatedCurrencies, meta.CreatedTags, meta.CreatedCategories = []string{}, []string{}, resolver.created
	metaJSON, _ := json.Marshal(meta)
	err = s.repo().MarkCompleted(ctx, importID, repo.Result{
		Processed: run.created + run.skipped + len(run.errs), Created: run.created, Skipped: run.skipped,
		Errors: len(run.errs), ErrorsJSON: string(errJSON), MetaJSON: string(metaJSON),
	}, now)
	if err != nil {
		return err
	}
	_ = s.Uploads.Discard(ctx, up)
	return nil
}

// maxRowErrors caps how many row errors an import report keeps.
const maxRowErrors = 200

// importRun tallies one Execute pass over the rows.
type importRun struct {
	s                Service
	accountID        int64
	unit             money.Unit
	resolver         *categoryResolver
	now              time.Time
	created, skipped int
	errs             []RowError
}

func (r *importRun) reject(n int, msg string) {
	if len(r.errs) < maxRowErrors {
		r.errs = append(r.errs, RowError{Row: n, Message: msg})
	}
}

// row imports one CSV row; n is its 1-based position.
func (r *importRun) row(ctx context.Context, row []string, mapping, options map[string]any, n int) {
	res := processImportRow(row, mapping, options, n)
	if res.err != "" {
		r.reject(n, res.err)
		return
	}
	amount, err := money.FromInput(res.amount, r.unit)
	if err != nil {
		r.reject(n, err.Error())
		return
	}
	added, err := r.s.repo().InsertRow(ctx, repo.Row{
		Type: res.typ, AccountID: r.accountID, CategoryID: r.resolver.resolve(res.category), Minor: amount.Minor(),
		Description: res.desc, Date: res.date, Status: "confirmed", DedupHash: dedupHash(res.date, res.amount, res.desc),
	}, r.now)
	switch {
	case err != nil:
		r.errs = append(r.errs, RowError{Row: n, Message: err.Error()})
	case added:
		r.created++
	default:
		r.skipped++
	}
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
				Description: res.desc, Status: status, Error: ptrIfSet(res.err), Category: ptrIfSet(res.category),
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
