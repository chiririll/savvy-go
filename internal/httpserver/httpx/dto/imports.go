package dto

import (
	"savvy-go/internal/importer"
)

type ImportFormats struct {
	DateFormat   string `json:"dateFormat"`
	AmountFormat string `json:"amountFormat"`
	HasHeader    bool   `json:"hasHeader"`
	Delimiter    string `json:"delimiter"`
}

type ImportParse struct {
	Headers          []string       `json:"headers"`
	PreviewRows      [][]string     `json:"previewRows"`
	TotalRows        *int           `json:"totalRows"`
	DetectedFormats  ImportFormats  `json:"detectedFormats"`
	SuggestedMapping map[string]int `json:"suggestedMapping"`
}

type ImportRowError struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

type ImportResult struct {
	Created           int              `json:"created"`
	SkippedDuplicates int              `json:"skippedDuplicates"`
	Errors            []ImportRowError `json:"errors"`
	CreatedCurrencies []string         `json:"createdCurrencies"`
	CreatedTags       []string         `json:"createdTags"`
	CreatedCategories []string         `json:"createdCategories"`
}

type Import struct {
	ImportID      string        `json:"importId"`
	Status        string        `json:"status"`
	TotalRows     *int          `json:"totalRows"`
	ProcessedRows int           `json:"processedRows"`
	Created       int           `json:"created"`
	Skipped       int           `json:"skipped"`
	Errors        int           `json:"errors"`
	Message       *string       `json:"message"`
	Parse         *ImportParse  `json:"parse"`
	Result        *ImportResult `json:"result"`
}

// NewImport renames the parse and result details of im to wire keys.
func NewImport(im importer.Import) Import {
	out := Import{
		ImportID: im.ID, Status: im.Status, TotalRows: im.TotalRows, ProcessedRows: im.ProcessedRows,
		Created: im.CreatedCount, Skipped: im.SkippedCount, Errors: im.ErrorCount, Message: im.Message,
	}
	meta := im.Meta
	if im.Status == "parsed" || im.Status == "importing" || im.Status == "completed" {
		f := meta.DetectedFormats
		out.Parse = &ImportParse{
			Headers:     orEmpty(meta.Headers),
			PreviewRows: orEmpty(meta.PreviewRows),
			TotalRows:   im.TotalRows,
			DetectedFormats: ImportFormats{
				DateFormat: f.DateFormat, AmountFormat: f.AmountFormat, HasHeader: f.HasHeader, Delimiter: f.Delimiter,
			},
			SuggestedMapping: meta.SuggestedMapping,
		}
	}
	if im.Status == "completed" {
		out.Result = &ImportResult{
			Created:           im.CreatedCount,
			SkippedDuplicates: im.SkippedCount,
			Errors:            Map(im.Errors, func(e importer.RowError) ImportRowError { return ImportRowError(e) }),
			CreatedCurrencies: orEmpty(meta.CreatedCurrencies),
			CreatedTags:       orEmpty(meta.CreatedTags),
			CreatedCategories: orEmpty(meta.CreatedCategories),
		}
	}
	return out
}

// orEmpty makes a nil slice serialize as [].
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

type ImportPreviewRow struct {
	Row         int     `json:"row"`
	Date        string  `json:"date"`
	Type        string  `json:"type"`
	Amount      Number  `json:"amount"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Error       *string `json:"error"`
	Category    *string `json:"category"`
	// The importer does not look for duplicates, tags or warnings yet; the
	// client still reads these keys.
	Tags        []string `json:"tags"`
	DuplicateOf *int64   `json:"duplicateOf"`
	Warnings    []string `json:"warnings"`
}

type ImportPreviewCategory struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Count   int    `json:"count"`
	MatchID *int64 `json:"matchId"`
}

type ImportPreviewSummary struct {
	WillCreate         int                     `json:"willCreate"`
	WillSkip           int                     `json:"willSkip"`
	HasErrors          int                     `json:"hasErrors"`
	TotalRows          *int                    `json:"totalRows"`
	Sampled            int                     `json:"sampled"`
	CurrenciesToCreate []string                `json:"currenciesToCreate"`
	TagsToCreate       []string                `json:"tagsToCreate"`
	CategoriesToCreate []string                `json:"categoriesToCreate"`
	Categories         []ImportPreviewCategory `json:"categories"`
}

type ImportPreview struct {
	PreviewTransactions []ImportPreviewRow   `json:"previewTransactions"`
	Summary             ImportPreviewSummary `json:"summary"`
}

func NewImportPreview(p importer.PreviewResult) ImportPreview {
	s := p.Summary
	return ImportPreview{
		PreviewTransactions: Map(p.Transactions, func(r importer.PreviewRow) ImportPreviewRow {
			return ImportPreviewRow{
				Row: r.Row, Date: r.Date, Type: r.Type, Amount: Number(r.Amount), Description: r.Description,
				Status: r.Status, Error: r.Error, Category: r.Category,
				Tags: []string{}, Warnings: []string{},
			}
		}),
		Summary: ImportPreviewSummary{
			WillCreate: s.WillCreate, WillSkip: s.WillSkip, HasErrors: s.HasErrors,
			TotalRows: s.TotalRows, Sampled: s.Sampled,
			CurrenciesToCreate: []string{}, TagsToCreate: []string{},
			CategoriesToCreate: orEmpty(s.CategoriesToCreate),
			Categories: Map(s.Categories, func(c importer.PreviewCategory) ImportPreviewCategory {
				return ImportPreviewCategory(c)
			}),
		},
	}
}
