package ledger

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

type Category struct {
	ID                int64
	Name, Type        string
	Icon, Color       *string
	IsDefault         bool
	TransactionsCount int
}

func categoryOf(id int64, name, typ string, icon, color db.NString, isDefault, count int64) Category {
	return Category{
		ID:                id,
		Name:              name,
		Type:              typ,
		Icon:              optString(icon),
		Color:             optString(color),
		IsDefault:         isDefault != 0,
		TransactionsCount: int(count),
	}
}

// Categories lists categories, of one type when typ is not empty.
func (r Repo) Categories(ctx context.Context, typ string) ([]Category, error) {
	rows, err := r.q().ListCategories(ctx, db.Narg(typ))
	if err != nil {
		return nil, err
	}
	out := make([]Category, len(rows))
	for i, x := range rows {
		out[i] = categoryOf(x.ID, x.Name, x.Type, x.Icon, x.Color, x.IsDefault, x.Count)
	}
	return out, nil
}

func (r Repo) Category(ctx context.Context, id int64) (*Category, error) {
	x, err := r.q().GetCategory(ctx, id)
	if err != nil {
		return nilIfMissing[Category](err)
	}
	c := categoryOf(x.ID, x.Name, x.Type, x.Icon, x.Color, x.IsDefault, x.Count)
	return &c, nil
}

func (r Repo) ClearDefaultCategory(ctx context.Context, typ string) error {
	return r.q().ClearDefaultCategory(ctx, typ)
}

func (r Repo) InsertCategory(ctx context.Context, c Category, now time.Time) (int64, error) {
	res, err := r.q().InsertCategory(ctx, sqlc.InsertCategoryParams{
		Name: c.Name, Type: c.Type, Icon: db.NullString(c.Icon), Color: db.NullString(c.Color),
		IsDefault: db.BoolInt(c.IsDefault), CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r Repo) UpdateCategory(ctx context.Context, id int64, c Category, now time.Time) error {
	return r.q().UpdateCategory(ctx, sqlc.UpdateCategoryParams{
		Name: c.Name, Type: c.Type, Icon: db.NullString(c.Icon), Color: db.NullString(c.Color),
		IsDefault: db.BoolInt(c.IsDefault), UpdatedAt: db.TimeNS(now), ID: id,
	})
}

// ReplaceCategory moves a category's transactions to another and deletes it.
func (r Repo) ReplaceCategory(ctx context.Context, id, successor int64) error {
	return store.Tx(ctx, r.DB, func(tx store.DB) error {
		q := db.Q(tx)
		if err := q.ReassignCategoryTransactions(ctx, sqlc.ReassignCategoryTransactionsParams{
			CategoryID: db.NI(successor), CategoryID_2: db.NI(id),
		}); err != nil {
			return err
		}
		return q.DeleteCategory(ctx, id)
	})
}

func (r Repo) DeleteCategory(ctx context.Context, id int64) error {
	return r.q().DeleteCategory(ctx, id)
}

func (r Repo) CountCategoriesOfType(ctx context.Context, typ string) int64 {
	n, _ := r.q().CountCategoriesByType(ctx, typ)
	return n
}

func (r Repo) SetCategoryDefault(ctx context.Context, id int64, now time.Time) error {
	return r.q().SetCategoryDefault(ctx, sqlc.SetCategoryDefaultParams{UpdatedAt: db.TimeNS(now), ID: id})
}

// CategoryAmount is one currency's share of a category's transactions.
type CategoryAmount struct {
	Count    int
	Minor    int64
	Decimals int
	Rate     decimal.Decimal
}

// CategoryStatistics lists, per currency, the category's confirmed
// transactions between start and end ("" for no bound).
func (r Repo) CategoryStatistics(ctx context.Context, id int64, start, end string) []CategoryAmount {
	rows, _ := r.q().CategoryStatistics(ctx, sqlc.CategoryStatisticsParams{
		CategoryID: db.NI(id), StartDate: db.Narg(start), EndDate: db.Narg(end),
	})
	out := make([]CategoryAmount, len(rows))
	for i, x := range rows {
		out[i] = CategoryAmount{Count: int(x.Cnt), Minor: x.Total, Decimals: int(x.Decimals), Rate: x.Rate}
	}
	return out
}

type Tag struct {
	ID                int64
	Name              string
	TransactionsCount int
	CreatedAt         *time.Time
}

func tagOf(id int64, name string, created db.NString, count int64) Tag {
	return Tag{ID: id, Name: name, TransactionsCount: int(count), CreatedAt: optTime(created)}
}

func (r Repo) Tags(ctx context.Context) ([]Tag, error) {
	rows, err := r.q().ListTags(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Tag, len(rows))
	for i, x := range rows {
		out[i] = tagOf(x.ID, x.Name, x.CreatedAt, x.Count)
	}
	return out, nil
}

func (r Repo) Tag(ctx context.Context, id int64) (*Tag, error) {
	x, err := r.q().GetTag(ctx, id)
	if err != nil {
		return nilIfMissing[Tag](err)
	}
	t := tagOf(x.ID, x.Name, x.CreatedAt, x.Count)
	return &t, nil
}

func (r Repo) InsertTag(ctx context.Context, name string, now time.Time) (int64, error) {
	res, err := r.q().
		InsertTag(ctx, sqlc.InsertTagParams{Name: name, CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now)})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r Repo) RenameTag(ctx context.Context, id int64, name string, now time.Time) error {
	return r.q().UpdateTag(ctx, sqlc.UpdateTagParams{Name: name, UpdatedAt: db.TimeNS(now), ID: id})
}

func (r Repo) DeleteTag(ctx context.Context, id int64) error { return r.q().DeleteTag(ctx, id) }
