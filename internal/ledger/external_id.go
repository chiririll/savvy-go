package ledger

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// MaxExternalIDLen is the longest external id accepted.
const MaxExternalIDLen = 255

// The sources of an external id. The server sets the source from how the
// request came in; a client sends only the id. A repeat of the same source and
// id is refused, so a retry on one channel adds nothing; the same id from
// another channel is stored and reported (see ExternalMatch).
const (
	SourceUI  = "ui"  // a signed-in browser session
	SourceAPI = "api" // a personal API token
	SourceMCP = "mcp" // an MCP client's token
	// SourceCSV is the CSV import's. It is not a source a request can have.
	SourceCSV = "csv"
)

// ExternalSources are the sources a request can have.
var ExternalSources = []string{SourceAPI, SourceMCP, SourceUI}

// ExternalMatch is a transaction that has the same external id under another
// source: possibly the same event entered twice, possibly not.
type ExternalMatch struct {
	Source        string
	TransactionID int64
}

// ErrExternalIDExists is the error of creating a transaction whose external id
// is already stored under the same source.
var ErrExternalIDExists = errors.New("a transaction with this external id exists")

// ExternalIDExistsError says which transaction already has the external id.
type ExternalIDExistsError struct {
	Source        string
	ExternalID    string
	TransactionID int64
}

func (e *ExternalIDExistsError) Error() string {
	return fmt.Sprintf("a transaction with external id %q exists (id %d)", e.ExternalID, e.TransactionID)
}
func (e *ExternalIDExistsError) Is(t error) bool { return t == ErrExternalIDExists }

// cleanExternalID checks the source and id of a new transaction and trims the
// id. Neither given is fine; an id without a known source is not.
func cleanExternalID(source, id *string) (*string, *string, error) {
	if id == nil {
		return nil, nil, nil
	}
	if source == nil || !slices.Contains(ExternalSources, *source) {
		return nil, nil, fmt.Errorf("external_id needs a request source (ui, api or mcp)")
	}
	v := strings.TrimSpace(*id)
	switch {
	case v == "":
		return nil, nil, fmt.Errorf("external_id must not be empty")
	case len(v) > MaxExternalIDLen:
		return nil, nil, fmt.Errorf("external_id is longer than %d characters", MaxExternalIDLen)
	}
	return source, &v, nil
}

// externalMatches finds the transactions that have tx's id under another
// source. A failed lookup gives none: it is only a warning.
func (s Transactions) externalMatches(ctx context.Context, tx *Transaction) []ExternalMatch {
	if tx.ExternalID == nil || tx.ExternalSource == nil {
		return nil
	}
	found, err := s.repo().OtherSourcesWithExternalID(ctx, *tx.ExternalSource, *tx.ExternalID)
	if err != nil {
		return nil
	}
	out := make([]ExternalMatch, len(found))
	for i, m := range found {
		out[i] = ExternalMatch{Source: m.Source, TransactionID: m.TransactionID}
	}
	return out
}
