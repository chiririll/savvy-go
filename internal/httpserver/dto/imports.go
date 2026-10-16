package dto

import (
	"savvy-go/internal/importer"
)

type ImportFormats struct {
	DateFormat   any  `json:"dateFormat"`
	AmountFormat any  `json:"amountFormat"`
	HasHeader    bool `json:"hasHeader"`
	Delimiter    any  `json:"delimiter"`
}

type ImportParse struct {
	Headers          any           `json:"headers"`
	PreviewRows      any           `json:"previewRows"`
	TotalRows        *int          `json:"totalRows"`
	DetectedFormats  ImportFormats `json:"detectedFormats"`
	SuggestedMapping any           `json:"suggestedMapping"`
}

type ImportResult struct {
	Created           int `json:"created"`
	SkippedDuplicates int `json:"skippedDuplicates"`
	Errors            any `json:"errors"`
	CreatedCurrencies any `json:"createdCurrencies"`
	CreatedTags       any `json:"createdTags"`
	CreatedCategories any `json:"createdCategories"`
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

// NewImport reads the parse/result details from im.Meta, which the domain
// stores as JSON with snake_case keys.
func NewImport(im importer.Import) Import {
	out := Import{
		ImportID: im.ID, Status: im.Status, TotalRows: im.TotalRows, ProcessedRows: im.ProcessedRows,
		Created: im.CreatedCount, Skipped: im.SkippedCount, Errors: im.ErrorCount, Message: im.Message,
	}
	parsed := im.Status == "parsed" || im.Status == "importing" || im.Status == "completed"
	if parsed && im.Meta != nil {
		formats, _ := im.Meta["detected_formats"].(map[string]any)
		hasHeader, _ := formats["has_header"].(bool)
		out.Parse = &ImportParse{
			Headers:     im.Meta["headers"],
			PreviewRows: im.Meta["preview_rows"],
			TotalRows:   im.TotalRows,
			DetectedFormats: ImportFormats{
				DateFormat: formats["date_format"], AmountFormat: formats["amount_format"],
				HasHeader: hasHeader, Delimiter: formats["delimiter"],
			},
			SuggestedMapping: im.Meta["suggested_mapping"],
		}
	}
	if im.Status == "completed" {
		out.Result = &ImportResult{
			Created:           im.CreatedCount,
			SkippedDuplicates: im.SkippedCount,
			Errors:            im.Errors,
			CreatedCurrencies: im.Meta["created_currencies"],
			CreatedTags:       im.Meta["created_tags"],
			CreatedCategories: im.Meta["created_categories"],
		}
	}
	return out
}
