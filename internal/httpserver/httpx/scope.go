package httpx

import (
	"context"
	"net/http"

	"savvy-go/internal/automation"
	"savvy-go/internal/importer"
	"savvy-go/internal/ledger"
	"savvy-go/internal/recurring"
	"savvy-go/internal/reports"
	"savvy-go/internal/settings"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
)

type ctxKey int

const ctxSpace ctxKey = iota

// SpaceScope is the space a request works in: the caller's role there and
// the ledger services bound to its database.
type SpaceScope struct {
	ID         int64
	Role       string
	DB         store.DB
	Settings   settings.Store
	Currencies ledger.Currencies
	Accounts   ledger.Accounts
	Categories ledger.Categories
	Tags       ledger.Tags
	Txs        ledger.Transactions
	Debts      ledger.Debts
	Recurring  recurring.Service
	Budgets    ledger.Budgets
	Automation automation.Service
	Reports    reports.Service
	Imports    importer.Service
}

// WithScope returns ctx carrying the space of a request.
func WithScope(ctx context.Context, scope *SpaceScope) context.Context {
	return context.WithValue(ctx, ctxSpace, scope)
}

// Scope is the space of a request that passed withSpace.
func Scope(r *http.Request) *SpaceScope {
	scope, _ := r.Context().Value(ctxSpace).(*SpaceScope)
	return scope
}

// Clone copies *p, so decoding into the copy leaves the original alone.
func Clone[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

// CurrentSpace is the registry entry of the request's space; it answers 404
// and returns nil when the space is gone.
func CurrentSpace(w http.ResponseWriter, r *http.Request, svc spaces.Service) *spaces.Space {
	space, err := svc.Get(r.Context(), Scope(r).ID)
	if err != nil || space == nil {
		WriteMessage(w, http.StatusNotFound, "Space not found.")
		return nil
	}
	return space
}
