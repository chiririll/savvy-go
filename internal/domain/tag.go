package domain

import (
	"context"
	"database/sql"
	"time"
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
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, name, created_at,
			(SELECT COUNT(*) FROM transaction_tag tt WHERE tt.tag_id = tags.id)
		FROM tags ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tag
	for rows.Next() {
		t, err := scanTag(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s Tags) ByID(ctx context.Context, id int64) (*Tag, error) {
	t, err := scanTag(s.DB.QueryRowContext(ctx, `
		SELECT id, name, created_at,
			(SELECT COUNT(*) FROM transaction_tag tt WHERE tt.tag_id = tags.id)
		FROM tags WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

func (s Tags) Create(ctx context.Context, name string) (*Tag, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.DB.ExecContext(ctx, `INSERT INTO tags (name, created_at, updated_at) VALUES (?,?,?)`, name, now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

func (s Tags) Update(ctx context.Context, id int64, name string) (*Tag, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.ExecContext(ctx, `UPDATE tags SET name=?, updated_at=? WHERE id=?`, name, now, id)
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Tags) Delete(ctx context.Context, id int64) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, id)
	return err
}

func scanTag(row interface{ Scan(...any) error }) (Tag, error) {
	var t Tag
	var created sql.NullString
	err := row.Scan(&t.ID, &t.Name, &created, &t.TransactionsCount)
	if tm, ok := parseNullTime(created); ok {
		t.CreatedAt = &tm
	}
	return t, err
}
