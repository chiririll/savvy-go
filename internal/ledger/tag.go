package ledger

import (
	"context"
	"time"

	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/store"
)

type Tag struct {
	ID                int64
	Name              string
	TransactionsCount int
	CreatedAt         *time.Time
}

type Tags struct{ DB store.DB }

func (s Tags) repo() repo.Repo { return repo.Repo{DB: s.DB} }

func (s Tags) All(ctx context.Context) ([]Tag, error) {
	rows, err := s.repo().Tags(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Tag, len(rows))
	for i, r := range rows {
		out[i] = tagFrom(r)
	}
	return out, nil
}

func (s Tags) ByID(ctx context.Context, id int64) (*Tag, error) {
	r, err := s.repo().Tag(ctx, id)
	if r == nil || err != nil {
		return nil, err
	}
	t := tagFrom(*r)
	return &t, nil
}

func (s Tags) Create(ctx context.Context, name string) (*Tag, error) {
	id, err := s.repo().InsertTag(ctx, name, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Tags) Update(ctx context.Context, id int64, name string) (*Tag, error) {
	if err := s.repo().RenameTag(ctx, id, name, time.Now().UTC()); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Tags) Delete(ctx context.Context, id int64) error {
	return s.repo().DeleteTag(ctx, id)
}

func tagFrom(r repo.Tag) Tag {
	return Tag{ID: r.ID, Name: r.Name, TransactionsCount: r.TransactionsCount, CreatedAt: r.CreatedAt}
}
