// Package recurringrepo stores recurring rules and the occurrences they made
// in a space database. A rule is a template (the transaction to create) and a
// schedule (when), both JSON that this package never reads: package recurring
// validates them and decides when a rule is due.
package recurring

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

type Repo struct{ DB store.DB }

func (r Repo) q() *sqlc.Queries { return db.Q(r.DB) }

// Row is a rule. LastDue is the latest due date handled, whatever became of
// its transaction; LastRun the latest one that was confirmed. Both are empty
// for a rule that made nothing yet. PendingDue is the due date of its open
// occurrence, if it has one.
type Row struct {
	ID                           int64
	Template                     string
	Schedule                     string
	IsActive                     bool
	Error                        *string
	LastDue, LastRun, PendingDue *string
	CreatedAt, UpdatedAt         *time.Time
}

// Fields is the writable part of a rule. Error is the reason the rule is
// stopped, nil for a rule that is fine.
type Fields struct {
	Template, Schedule string
	IsActive           bool
	Error              *string
}

func rowOf(x sqlc.ListRecurringRow) Row {
	r := Row{ID: x.ID, Template: x.Template, Schedule: x.Schedule, IsActive: x.IsActive != 0}
	if x.Error.Valid {
		r.Error = &x.Error.String
	}
	if v, ok := x.LastDue.(string); ok {
		r.LastDue = &v
	}
	if v, ok := x.LastRun.(string); ok {
		r.LastRun = &v
	}
	if v, ok := x.PendingDue.(string); ok {
		r.PendingDue = &v
	}
	if t, ok := db.ParseTime(x.CreatedAt); ok {
		r.CreatedAt = &t
	}
	if t, ok := db.ParseTime(x.UpdatedAt); ok {
		r.UpdatedAt = &t
	}
	return r
}

// List lists the rules by id; activeOnly keeps the active ones.
func (r Repo) List(ctx context.Context, activeOnly bool) ([]Row, error) {
	rows, err := r.q().ListRecurring(ctx, db.Flag(activeOnly))
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
		Template: f.Template, Schedule: f.Schedule, IsActive: db.BoolInt(f.IsActive), Error: db.NullString(f.Error),
		CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r Repo) Update(ctx context.Context, id int64, f Fields, now time.Time) error {
	return r.q().UpdateRecurring(ctx, sqlc.UpdateRecurringParams{
		Template: f.Template, Schedule: f.Schedule, IsActive: db.BoolInt(f.IsActive), Error: db.NullString(f.Error),
		UpdatedAt: db.TimeNS(now), ID: id,
	})
}

// Stop deactivates a rule and records why.
func (r Repo) Stop(ctx context.Context, id int64, reason string, now time.Time) error {
	return r.q().StopRecurring(ctx, sqlc.StopRecurringParams{Error: db.NS(reason), UpdatedAt: db.TimeNS(now), ID: id})
}

// Delete removes a rule and its instances. The transactions it made stay.
func (r Repo) Delete(ctx context.Context, id int64) error {
	return r.q().DeleteRecurring(ctx, id)
}

// Instance is a handled due date. TransactionID is nil once the transaction
// was deleted.
type Instance struct {
	DueDate       string
	TransactionID *int64
}

// AddInstance records that a rule handled a due date with a transaction. It
// reports false, and writes nothing, when the rule had already handled it.
func (r Repo) AddInstance(ctx context.Context, rule int64, due string, txID int64) (bool, error) {
	res, err := r.q().InsertRecurringInstance(ctx, sqlc.InsertRecurringInstanceParams{
		RecurringTransactionID: rule, DueDate: due, TransactionID: db.NI(txID),
	})
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// Instances lists the due dates a rule handled, oldest first.
func (r Repo) Instances(ctx context.Context, rule int64) ([]Instance, error) {
	rows, err := r.q().ListRecurringInstances(ctx, rule)
	if err != nil {
		return nil, err
	}
	out := make([]Instance, len(rows))
	for i, x := range rows {
		out[i] = Instance{DueDate: x.DueDate}
		if x.TransactionID.Valid {
			out[i].TransactionID = &x.TransactionID.Int64
		}
	}
	return out, nil
}

// Pending is the open occurrence of a rule; nil when it has none.
func (r Repo) Pending(ctx context.Context, rule int64) (*Instance, error) {
	x, err := r.q().GetPendingRecurringInstance(ctx, rule)
	if err == db.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &Instance{DueDate: x.DueDate, TransactionID: &x.TransactionID.Int64}, nil
}

// OccurrenceOf is the rule and due date a transaction was made for; nil for a
// transaction no rule made.
func (r Repo) OccurrenceOf(ctx context.Context, txID int64) (rule int64, due string, ok bool, err error) {
	x, err := r.q().GetRecurringInstanceOf(ctx, db.NI(txID))
	if err == db.ErrNotFound {
		return 0, "", false, nil
	}
	if err != nil {
		return 0, "", false, err
	}
	return x.RecurringTransactionID, x.DueDate, true, nil
}

// MoveInstance changes the due date of a handled occurrence. It reports false
// when the rule already handled the new date.
func (r Repo) MoveInstance(ctx context.Context, rule int64, from, to string) (bool, error) {
	taken, err := r.q().CountRecurringInstancesOn(ctx,
		sqlc.CountRecurringInstancesOnParams{RecurringTransactionID: rule, DueDate: to})
	if err != nil || taken > 0 {
		return false, err
	}
	res, err := r.q().MoveRecurringInstance(ctx,
		sqlc.MoveRecurringInstanceParams{DueDate: to, RecurringTransactionID: rule, DueDate_2: from})
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
