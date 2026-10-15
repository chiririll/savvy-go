package ledger

import (
	"context"

	"savvy-go/internal/store"
)

// Related loads the accounts and categories that a page of rows points at,
// each once, so hydrating a list costs one lookup per distinct account or
// category instead of one per row. It lives for one call.
type Related struct {
	ctx        context.Context
	accts      Accounts
	cats       Categories
	accounts   map[int64]*Account
	categories map[int64]*Category
}

// NewRelated starts an empty cache over one space database.
func NewRelated(ctx context.Context, sqlDB store.DB) *Related {
	return &Related{
		ctx: ctx, accts: Accounts{DB: sqlDB}, cats: Categories{DB: sqlDB},
		accounts: map[int64]*Account{}, categories: map[int64]*Category{},
	}
}

// Account is the account with id, nil when it does not exist or fails to load.
func (r *Related) Account(id int64) *Account {
	if a, ok := r.accounts[id]; ok {
		return a
	}
	a, _ := r.accts.ByID(r.ctx, id)
	r.accounts[id] = a
	return a
}

// OptionalAccount is Account for an optional reference.
func (r *Related) OptionalAccount(id *int64) *Account {
	if id == nil {
		return nil
	}
	return r.Account(*id)
}

// Category is the category an optional reference points at.
func (r *Related) Category(id *int64) *Category {
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
