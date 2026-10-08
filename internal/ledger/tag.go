package ledger

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

type Tag struct {
	ID                int64
	Name              string
	TransactionsCount int
	CreatedAt         *time.Time
}

type Tags struct{ DB store.DB }

func (s Tags) All(ctx context.Context) ([]Tag, error) {
	rows, err := db.Q(s.DB).ListTags(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Tag, 0, len(rows))
	for _, r := range rows {
		out = append(out, TagFromRow(r.ID, r.Name, r.CreatedAt, r.Count))
	}
	return out, nil
}

func (s Tags) ByID(ctx context.Context, id int64) (*Tag, error) {
	r, err := db.Q(s.DB).GetTag(ctx, id)
	if err == db.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t := TagFromRow(r.ID, r.Name, r.CreatedAt, r.Count)
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

// TagFromRow builds a tag from the columns of a tag row.
func TagFromRow(id int64, name string, created db.NString, count int64) Tag {
	t := Tag{ID: id, Name: name, TransactionsCount: int(count)}
	if tm, ok := db.ParseTime(created); ok {
		t.CreatedAt = &tm
	}
	return t
}

func tagFromSQL(id int64, name string, created db.NString, count int64) Tag {
	return TagFromRow(id, name, created, count)
}
