package domain

import (
	"context"
	"database/sql"
	"time"

	"github.com/chiririll/savvy-plus/internal/db"
	"github.com/chiririll/savvy-plus/internal/db/sqlc"
)

type Tag struct {
	ID                int64
	Name              string
	TransactionsCount int
	CreatedAt         *time.Time
}

func (t Tag) JSON() map[string]any {
	var created any
	if t.CreatedAt != nil {
		created = t.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return map[string]any{
		"id":                 t.ID,
		"name":               t.Name,
		"transactionsCount":  t.TransactionsCount,
		"createdAt":          created,
	}
}

type Tags struct{ DB *sql.DB }

func (s Tags) All(ctx context.Context) ([]Tag, error) {
	rows, err := db.Q(s.DB).ListTags(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Tag, 0, len(rows))
	for _, r := range rows {
		out = append(out, tagFromList(r.ID, r.Name, r.CreatedAt, r.Count))
	}
	return out, nil
}

func (s Tags) ByID(ctx context.Context, id int64) (*Tag, error) {
	r, err := db.Q(s.DB).GetTag(ctx, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t := tagFromList(r.ID, r.Name, r.CreatedAt, r.Count)
	return &t, nil
}

func (s Tags) Create(ctx context.Context, name string) (*Tag, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Q(s.DB).InsertTag(ctx, sqlc.InsertTagParams{Name: name, CreatedAt: db.NS(now), UpdatedAt: db.NS(now)})
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

func (s Tags) Update(ctx context.Context, id int64, name string) (*Tag, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	if err := db.Q(s.DB).UpdateTag(ctx, sqlc.UpdateTagParams{Name: name, UpdatedAt: db.NS(now), ID: id}); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Tags) Delete(ctx context.Context, id int64) error {
	return db.Q(s.DB).DeleteTag(ctx, id)
}

func tagFromList(id int64, name string, created sql.NullString, count int64) Tag {
	t := Tag{ID: id, Name: name, TransactionsCount: int(count)}
	if tm, ok := parseNullTime(created); ok {
		t.CreatedAt = &tm
	}
	return t
}

func tagFromSQL(id int64, name string, created sql.NullString, count int64) Tag {
	return tagFromList(id, name, created, count)
}
