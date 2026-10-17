// Package recurringrepo stores recurring transaction templates in a space
// database. Package recurring decides when they are due and what they plan.
package recurring

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/money"
	"savvy-go/internal/store"
)

type Repo struct{ DB store.DB }

func (r Repo) q() *sqlc.Queries { return db.Q(r.DB) }

// Row is a template. Amount is in the currency of its account and ToAmount in
// the currency of the to-account.
type Row struct {
	ID                      int64
	Type, Frequency         string
	AccountID               int64
	ToAccountID, CategoryID *int64
	Amount                  money.Money
	ToAmount                *money.Money
	IsEstimated, IsActive   bool
	Description             *string
	Interval                int
	DayOfWeek, DayOfMonth   *int
	StartDate, NextRunDate  string
	EndDate, LastRunDate    *string
}

// Fields is the writable part of a template.
type Fields struct {
	Type, Frequency         string
	AccountID               int64
	ToAccountID, CategoryID *int64
	Amount                  money.Money
	ToAmount                *money.Money
	IsEstimated, IsActive   bool
	Description             *string
	Interval                int
	DayOfWeek, DayOfMonth   *int
	StartDate, NextRunDate  string
	EndDate                 *string
}

func rowOf(x sqlc.ListRecurringRow) Row {
	r := Row{
		ID: x.ID, Type: x.Type, Frequency: x.Frequency, AccountID: x.AccountID,
		Amount:      money.New(x.Amount, money.Unit{ID: x.CurrencyID, Decimals: int(x.Decimals)}),
		ToAmount:    db.MoneyFromNull(x.ToAmount, money.Unit{ID: x.ToCurrencyID, Decimals: int(x.ToDecimals)}),
		IsEstimated: x.IsEstimated != 0, IsActive: x.IsActive != 0, Interval: int(x.Interval),
		StartDate: x.StartDate, NextRunDate: x.NextRunDate,
	}
	if x.ToAccountID.Valid {
		r.ToAccountID = &x.ToAccountID.Int64
	}
	if x.CategoryID.Valid {
		r.CategoryID = &x.CategoryID.Int64
	}
	if x.Description.Valid {
		r.Description = &x.Description.String
	}
	if x.DayOfWeek.Valid {
		v := int(x.DayOfWeek.Int64)
		r.DayOfWeek = &v
	}
	if x.DayOfMonth.Valid {
		v := int(x.DayOfMonth.Int64)
		r.DayOfMonth = &v
	}
	if x.EndDate.Valid {
		r.EndDate = &x.EndDate.String
	}
	if x.LastRunDate.Valid {
		r.LastRunDate = &x.LastRunDate.String
	}
	return r
}

// List lists templates, newest first by the query's order; activeOnly keeps
// the active ones, limit caps the count.
func (r Repo) List(ctx context.Context, activeOnly bool, limit int) ([]Row, error) {
	rows, err := r.q().
		ListRecurring(ctx, sqlc.ListRecurringParams{ActiveOnly: db.Flag(activeOnly), Limit: int64(limit), Offset: 0})
	if err != nil {
		return nil, err
	}
	out := make([]Row, len(rows))
	for i, x := range rows {
		out[i] = rowOf(x)
	}
	return out, nil
}

// ByID returns nil for an unknown id.
func (r Repo) ByID(ctx context.Context, id int64) (*Row, error) {
	x, err := r.q().GetRecurring(ctx, id)
	if err == db.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row := rowOf(sqlc.ListRecurringRow(x))
	return &row, nil
}

func (r Repo) Insert(ctx context.Context, f Fields, now time.Time) (int64, error) {
	res, err := r.q().InsertRecurring(ctx, sqlc.InsertRecurringParams{
		Type: f.Type, AccountID: f.AccountID, ToAccountID: db.NullInt64(f.ToAccountID),
		CategoryID: db.NullInt64(f.CategoryID), Amount: f.Amount.Minor(),
		ToAmount: db.NullMoney(f.ToAmount), IsEstimated: db.BoolInt(f.IsEstimated),
		Description: db.NullString(f.Description), Frequency: f.Frequency, Interval: int64(f.Interval),
		DayOfWeek: db.NullInt(f.DayOfWeek), DayOfMonth: db.NullInt(f.DayOfMonth),
		StartDate: f.StartDate, EndDate: db.NullString(f.EndDate), NextRunDate: f.NextRunDate,
		IsActive: db.BoolInt(f.IsActive), CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r Repo) Update(ctx context.Context, id int64, f Fields, now time.Time) error {
	return r.q().UpdateRecurring(ctx, sqlc.UpdateRecurringParams{
		Type: f.Type, AccountID: f.AccountID, ToAccountID: db.NullInt64(f.ToAccountID),
		CategoryID: db.NullInt64(f.CategoryID), Amount: f.Amount.Minor(),
		ToAmount: db.NullMoney(f.ToAmount), IsEstimated: db.BoolInt(f.IsEstimated),
		Description: db.NullString(f.Description), Frequency: f.Frequency, Interval: int64(f.Interval),
		DayOfWeek: db.NullInt(f.DayOfWeek), DayOfMonth: db.NullInt(f.DayOfMonth),
		StartDate: f.StartDate, EndDate: db.NullString(f.EndDate), NextRunDate: f.NextRunDate,
		IsActive: db.BoolInt(f.IsActive), UpdatedAt: db.TimeNS(now), ID: id,
	})
}

// Delete removes a template and its pending occurrence.
func (r Repo) Delete(ctx context.Context, id int64) error {
	if err := r.q().DeletePendingForRecurring(ctx, db.NI(id)); err != nil {
		return err
	}
	return r.q().DeleteRecurring(ctx, id)
}

// Advance records a run on lastRun and moves the template to its next date.
func (r Repo) Advance(ctx context.Context, id int64, lastRun, next string, now time.Time) error {
	return r.q().AdvanceRecurring(ctx, sqlc.AdvanceRecurringParams{
		LastRunDate: db.NS(lastRun), NextRunDate: next, UpdatedAt: db.TimeNS(now), ID: id,
	})
}

// PendingTransactionID is the pending occurrence of a template; 0 when none.
func (r Repo) PendingTransactionID(ctx context.Context, id int64) (int64, error) {
	tx, err := r.q().GetPendingRecurringTx(ctx, db.NI(id))
	if err == db.ErrNotFound {
		return 0, nil
	}
	return tx, err
}

// SetTags replaces the tags of a template.
func (r Repo) SetTags(ctx context.Context, id int64, tagIDs []int64) error {
	if err := r.q().DeleteRecurringTags(ctx, id); err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		if err := r.q().InsertRecurringTag(ctx, sqlc.InsertRecurringTagParams{RecurringTransactionID: id, TagID: tagID}); err != nil {
			return err
		}
	}
	return nil
}

// Tag is a tag of a template.
type Tag struct {
	ID                int64
	Name              string
	TransactionsCount int
	CreatedAt         *time.Time
}

func (r Repo) Tags(ctx context.Context, id int64) ([]Tag, error) {
	rows, err := r.q().ListRecurringTags(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Tag, len(rows))
	for i, x := range rows {
		out[i] = Tag{ID: x.ID, Name: x.Name, TransactionsCount: int(x.TransactionsCount)}
		if t, ok := db.ParseTime(x.CreatedAt); ok {
			out[i].CreatedAt = &t
		}
	}
	return out, nil
}
