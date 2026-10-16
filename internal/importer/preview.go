package importer

import "github.com/shopspring/decimal"

// PreviewRow is one CSV row as it would be imported.
type PreviewRow struct {
	Row         int
	Date, Type  string
	Amount      decimal.Decimal
	Description string
	Status      string // new or error
	Error       *string
	Category    *string
}

// PreviewCategory is a category the file names, and the existing one it
// matches, if any.
type PreviewCategory struct {
	Name, Type string
	Count      int
	MatchID    *int64
}

type PreviewSummary struct {
	WillCreate, WillSkip, HasErrors int
	TotalRows                       *int
	Sampled                         int
	CategoriesToCreate              []string
	Categories                      []PreviewCategory
}

// PreviewResult is what an import would do, from a sample of its rows.
type PreviewResult struct {
	Transactions []PreviewRow
	Summary      PreviewSummary
}
