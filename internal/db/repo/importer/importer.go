// Package importerrepo stores the records of CSV imports in a space database
// and inserts the imported rows.
package importer

import (
	"context"
	"fmt"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

// Repo reads and writes import records of one space database.
type Repo struct{ DB store.DB }

// Import is one record; Mapping, Options, Errors and Meta are JSON text.
type Import struct {
	ID, Status                                string
	UserID                                    *int64
	UploadID                                  *string
	TotalRows                                 *int
	ProcessedRows, CreatedCount, SkippedCount int
	ErrorCount                                int
	Mapping, Options, Errors, Meta, Message   *string
}

func (r Repo) Insert(ctx context.Context, id string, userID int64, uploadID, status string, now time.Time) error {
	return db.Q(r.DB).InsertImport(ctx, sqlc.InsertImportParams{
		ID: id, UserID: db.NI(userID), UploadID: db.NS(uploadID), Status: status,
		CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
}

// ByID returns nil for an unknown id.
func (r Repo) ByID(ctx context.Context, id string) (*Import, error) {
	row, err := db.Q(r.DB).GetImport(ctx, id)
	if err == db.ErrNotFound {
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
	for _, f := range []struct {
		from db.NString
		to   **string
	}{{row.Mapping, &im.Mapping}, {row.Options, &im.Options}, {row.Errors, &im.Errors}, {row.Meta, &im.Meta},
		{row.Message, &im.Message}} {
		if f.from.Valid {
			s := f.from.String
			*f.to = &s
		}
	}
	return &im, nil
}

// MarkParsed records that the file was read: how many rows it has and the
// preview metadata.
func (r Repo) MarkParsed(ctx context.Context, id string, totalRows int, meta string, now time.Time) error {
	return db.Q(r.DB).MarkImportParsed(ctx, sqlc.MarkImportParsedParams{
		TotalRows: db.NI(int64(totalRows)), Meta: db.NS(meta), UpdatedAt: db.TimeNS(now), ID: id,
	})
}

func (r Repo) MarkImporting(ctx context.Context, id, mapping, options string, now time.Time) error {
	return db.Q(r.DB).MarkImportImporting(ctx, sqlc.MarkImportImportingParams{
		Mapping: db.NS(mapping), Options: db.NS(options), UpdatedAt: db.TimeNS(now), ID: id,
	})
}

// Result is how an import ended.
type Result struct {
	Processed, Created, Skipped, Errors int
	ErrorsJSON, MetaJSON                string
}

func (r Repo) MarkCompleted(ctx context.Context, id string, res Result, now time.Time) error {
	return db.Q(r.DB).MarkImportCompleted(ctx, sqlc.MarkImportCompletedParams{
		ProcessedRows: int64(res.Processed), CreatedCount: int64(res.Created), SkippedCount: int64(res.Skipped),
		ErrorCount: int64(res.Errors), Errors: db.NS(res.ErrorsJSON), Meta: db.NS(res.MetaJSON),
		UpdatedAt: db.TimeNS(now), ID: id,
	})
}

func (r Repo) Fail(ctx context.Context, id, message string, now time.Time) error {
	return db.Q(r.DB).FailImport(ctx, sqlc.FailImportParams{Message: db.NS(message), UpdatedAt: db.TimeNS(now), ID: id})
}

// Row is a transaction read from a file.
type Row struct {
	Type, Description, Date, Status, DedupHash string
	AccountID                                  int64
	CategoryID                                 *int64
	Minor                                      int64
}

// InsertRow adds the transaction unless one with the same dedup hash exists
// for the account, and reports whether it added it. The row is one payment on
// the account (negative for an expense) and one line of the same amount.
func (r Repo) InsertRow(ctx context.Context, row Row, now time.Time) (bool, error) {
	externalID := fmt.Sprintf("csv:%d:%s", row.AccountID, row.DedupHash)
	added := false
	err := store.Tx(ctx, r.DB, func(d store.DB) error {
		q := db.Q(d)
		if dup, err := q.ExistsTransactionExternalID(ctx, db.NS(externalID)); err != nil || dup != 0 {
			return err
		}
		unit, err := q.GetAccountUnit(ctx, row.AccountID)
		if err != nil {
			return err
		}
		res, err := q.InsertTransaction(ctx, sqlc.InsertTransactionParams{
			Type: row.Type, Description: db.NS(row.Description), Date: db.NS(row.Date), Status: row.Status,
			ExternalID: db.NS(externalID), CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
		})
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		amount := row.Minor
		if amount < 0 {
			amount = -amount
		}
		paid := amount
		if row.Type == "expense" {
			paid = -amount
		}
		if err := q.InsertPayment(ctx, sqlc.InsertPaymentParams{
			TransactionID: id, AccountID: row.AccountID, Amount: paid,
		}); err != nil {
			return err
		}
		added = true
		return q.InsertTransactionLine(ctx, sqlc.InsertTransactionLineParams{
			TransactionID: id, Seq: 0, CategoryID: db.NullInt64(row.CategoryID), CurrencyID: unit.ID, Amount: amount,
		})
	})
	return added && err == nil, err
}

// CountsByStatus is how many imports are in each status.
func (r Repo) CountsByStatus(ctx context.Context) (map[string]int, error) {
	rows, err := db.Q(r.DB).ImportCountsByStatus(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		out[row.Status] = int(row.Count)
	}
	return out, nil
}
