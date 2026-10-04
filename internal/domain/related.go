package domain

import (
	"context"
	"savvy-go/internal/store"
)

// related loads the accounts and categories that a page of rows points at,
// each once, so hydrating a list costs one lookup per distinct account or
// category instead of one per row. It lives for one call.
type related struct {
	ctx        context.Context
	accts      Accounts
	cats       Categories
	accounts   map[int64]*Account
	categories map[int64]*Category
}

func newRelated(ctx context.Context, sqlDB store.DB) *related {
	return &related{
		ctx: ctx, accts: Accounts{DB: sqlDB}, cats: Categories{DB: sqlDB},
		accounts: map[int64]*Account{}, categories: map[int64]*Category{},
	}
}

// account is the account with id, nil when it does not exist or fails to load.
func (r *related) account(id int64) *Account {
	if a, ok := r.accounts[id]; ok {
		return a
	}
	a, _ := r.accts.ByID(r.ctx, id)
	r.accounts[id] = a
	return a
}

// optionalAccount is account for an optional reference.
func (r *related) optionalAccount(id *int64) *Account {
	if id == nil {
		return nil
	}
	return r.account(*id)
}

// category is the category an optional reference points at.
func (r *related) category(id *int64) *Category {
	if id == nil {
		return nil
	}
	if c, ok := r.categories[*id]; ok {
		return c
	}
	c, _ := r.cats.ByID(r.ctx, *id)
	r.categories[*id] = c
	return c
}
