// Package audit reads and writes the audit_log table of one space database.
// Package internal/audit owns the rules (who may undo what); this one only
// moves rows.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

// Repo works on one space database or on a transaction on it.
type Repo struct{ DB store.DB }

func (r Repo) q() *sqlc.Queries { return db.Q(r.DB) }

// Row is one audit_log row. Before and After are JSON snapshots, nil when the
// entity did not exist before or after.
type Row struct {
	ID           int64
	At           time.Time
	Source       string
	UserID       *int64
	TokenID      *int64
	TokenLabel   string
	EntityType   string
	EntityID     *int64
	Op           string
	Before       json.RawMessage
	After        json.RawMessage
	BatchID      string
	RolledBackBy *int64
	Details      string
}

// New is a row to write.
type New struct {
	At         time.Time
	Source     string
	UserID     *int64
	TokenID    *int64
	TokenLabel string
	EntityType string
	EntityID   *int64
	Op         string
	Before     json.RawMessage
	After      json.RawMessage
	BatchID    string
	Details    string
}

func raw(b json.RawMessage) *string {
	if len(b) == 0 {
		return nil
	}
	s := string(b)
	return &s
}

func str(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Insert writes a row and returns its id.
func (r Repo) Insert(ctx context.Context, n New) (int64, error) {
	res, err := r.q().InsertAuditLog(ctx, sqlc.InsertAuditLogParams{
		At: db.FormatTime(n.At), Source: n.Source, UserID: db.NullInt64(n.UserID), TokenID: db.NullInt64(n.TokenID),
		TokenLabel: db.NullString(str(n.TokenLabel)), EntityType: n.EntityType, EntityID: db.NullInt64(n.EntityID),
		Op: n.Op, BeforeJson: db.NullString(raw(n.Before)), AfterJson: db.NullString(raw(n.After)),
		BatchID: db.NullString(str(n.BatchID)), Details: db.NullString(str(n.Details)),
	})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func rowOf(x sqlc.AuditLog) Row {
	at, _ := db.ParseTimeString(x.At)
	row := Row{
		ID: x.ID, At: at, Source: x.Source, TokenLabel: x.TokenLabel.String, EntityType: x.EntityType, Op: x.Op,
		BatchID: x.BatchID.String, Details: x.Details.String,
	}
	if x.UserID.Valid {
		row.UserID = &x.UserID.Int64
	}
	if x.TokenID.Valid {
		row.TokenID = &x.TokenID.Int64
	}
	if x.EntityID.Valid {
		row.EntityID = &x.EntityID.Int64
	}
	if x.RolledBackBy.Valid {
		row.RolledBackBy = &x.RolledBackBy.Int64
	}
	if x.BeforeJson.Valid {
		row.Before = json.RawMessage(x.BeforeJson.String)
	}
	if x.AfterJson.Valid {
		row.After = json.RawMessage(x.AfterJson.String)
	}
	return row
}

func rowsOf(list []sqlc.AuditLog) []Row {
	out := make([]Row, len(list))
	for i, x := range list {
		out[i] = rowOf(x)
	}
	return out
}

// Get returns nil for an unknown id.
func (r Repo) Get(ctx context.Context, id int64) (*Row, error) {
	x, err := r.q().GetAuditLog(ctx, id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	row := rowOf(x)
	return &row, nil
}

// Filter narrows a listing; a zero field is no filter.
type Filter struct {
	TokenID    int64
	UserID     int64
	EntityType string
	EntityID   int64
	Op         string
	Source     string
	BeforeID   int64 // rows with a smaller id: the next page
}

// List returns up to limit rows, newest first.
func (r Repo) List(ctx context.Context, f Filter, limit int) ([]Row, error) {
	if f.BeforeID == 0 {
		f.BeforeID = math.MaxInt64
	}
	if f.UserID != 0 {
		p := sqlc.ListAuditLogUserParams{BeforeID: f.BeforeID, UserID: db.NI(f.UserID), RowLimit: int64(limit)}
		if f.EntityType != "" {
			p.EntityType = db.NS(f.EntityType)
		}
		if f.Op != "" {
			p.Op = db.NS(f.Op)
		}
		if f.Source != "" {
			p.Source = db.NS(f.Source)
		}
		list, err := r.q().ListAuditLogUser(ctx, p)
		return rowsOf(list), err
	}
	p := sqlc.ListAuditLogParams{BeforeID: f.BeforeID, RowLimit: int64(limit)}
	if f.TokenID != 0 {
		p.TokenID = db.NI(f.TokenID)
	}
	if f.EntityType != "" {
		p.EntityType = db.NS(f.EntityType)
	}
	if f.EntityID != 0 {
		p.EntityID = db.NI(f.EntityID)
	}
	if f.Op != "" {
		p.Op = db.NS(f.Op)
	}
	if f.Source != "" {
		p.Source = db.NS(f.Source)
	}
	list, err := r.q().ListAuditLog(ctx, p)
	return rowsOf(list), err
}

// Batch lists the rows of one batch, newest first.
func (r Repo) Batch(ctx context.Context, batchID string) ([]Row, error) {
	list, err := r.q().ListAuditLogBatch(ctx, db.NS(batchID))
	return rowsOf(list), err
}

// FirstOverlap is the id of the first standing row after afterID that changed
// the same entity, or 0 when there is none.
func (r Repo) FirstOverlap(ctx context.Context, entityType string, entityID, afterID int64) (int64, error) {
	id, err := r.q().FirstAuditOverlap(ctx, sqlc.FirstAuditOverlapParams{
		EntityType: entityType, EntityID: db.NI(entityID), ID: afterID,
	})
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return id, nil
}

// Standing is a standing row that changed an entity.
type Standing struct{ ID, EntityID int64 }

// StandingAfter lists, oldest first, the standing rows after afterID that
// changed any of the entities of one type: FirstOverlap for many at once.
func (r Repo) StandingAfter(ctx context.Context, entityType string, entityIDs []int64, afterID int64) ([]Standing, error) {
	if len(entityIDs) == 0 {
		return nil, nil
	}
	ids := make([]sql.NullInt64, len(entityIDs))
	for i, id := range entityIDs {
		ids[i] = db.NI(id)
	}
	list, err := r.q().ListAuditStanding(ctx, sqlc.ListAuditStandingParams{EntityType: entityType, EntityIds: ids, ID: afterID})
	out := make([]Standing, len(list))
	for i, x := range list {
		out[i] = Standing{ID: x.ID, EntityID: x.EntityID.Int64}
	}
	return out, err
}

// MarkRolledBack records that row id was undone by row by. It reports whether
// the row was still standing.
func (r Repo) MarkRolledBack(ctx context.Context, id, by int64) (bool, error) {
	row, err := r.Get(ctx, id)
	if err != nil || row == nil || row.RolledBackBy != nil {
		return false, err
	}
	return true, r.q().MarkAuditRolledBack(ctx, sqlc.MarkAuditRolledBackParams{RolledBackBy: db.NI(by), ID: id})
}

// DeleteUpTo removes rows with an id up to maxID, all of them or only those of
// one token (tokenID 0 for all), and returns how many.
func (r Repo) DeleteUpTo(ctx context.Context, maxID, tokenID int64) (int64, error) {
	if tokenID != 0 {
		res, err := r.q().DeleteAuditLogOfTokenUpTo(ctx, sqlc.DeleteAuditLogOfTokenUpToParams{ID: maxID, TokenID: db.NI(tokenID)})
		if err != nil {
			return 0, err
		}
		return res.RowsAffected()
	}
	res, err := r.q().DeleteAuditLogUpTo(ctx, maxID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// MaxID is the id of the newest row, 0 when the log is empty.
func (r Repo) MaxID(ctx context.Context) (int64, error) {
	return r.q().MaxAuditLogID(ctx)
}
