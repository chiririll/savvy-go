package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/automation"
	"savvy-go/internal/importer"
	"savvy-go/internal/ledger"
	"savvy-go/internal/recurring"
	"savvy-go/internal/reports"
	"savvy-go/internal/settings"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
)

const ctxSpace ctxKey = 100

// spaceScope is the space a request works in: the caller's role there and
// the ledger services bound to its database.
type spaceScope struct {
	id         int64
	role       string
	db         store.DB
	settings   settings.Store
	currencies ledger.Currencies
	accounts   ledger.Accounts
	categories ledger.Categories
	tags       ledger.Tags
	txs        ledger.Transactions
	debts      ledger.Debts
	recurring  recurring.Service
	budgets    ledger.Budgets
	automation automation.Service
	reports    reports.Service
	imports    importer.Service
}

func (s *Server) newScope(id int64, role string, d store.DB) *spaceScope {
	txs := ledger.Transactions{DB: d}
	accounts := ledger.Accounts{DB: d}
	return &spaceScope{
		id:         id,
		role:       role,
		db:         d,
		settings:   settings.Store{DB: d, Space: true},
		currencies: ledger.Currencies{DB: d, Source: s.rates},
		accounts:   accounts,
		categories: ledger.Categories{DB: d},
		tags:       ledger.Tags{DB: d},
		txs:        txs,
		debts:      ledger.Debts{Accounts: accounts, Transactions: txs},
		recurring:  recurring.Service{DB: d, Txs: txs},
		budgets:    ledger.Budgets{DB: d},
		automation: automation.Service{DB: d, Txs: txs},
		reports:    reports.Service{DB: d, Loc: s.cfg.Location},
		imports:    importer.Service{DB: d, Uploads: s.uploads, Txs: txs},
	}
}

// sp is the space of a request that passed withSpace.
func sp(r *http.Request) *spaceScope {
	scope, _ := r.Context().Value(ctxSpace).(*spaceScope)
	return scope
}

// withSpace resolves the space named by {space} and the caller's role in
// it. Membership is checked before availability, so a space the caller does
// not belong to is a 404 whether or not it is broken (P42).
func (s *Server) withSpace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFrom(r)
		if u == nil {
			writeMessage(w, http.StatusUnauthorized, "Unauthenticated.")
			return
		}
		ctx := r.Context()
		id, err := strconv.ParseInt(chi.URLParam(r, "space"), 10, 64)
		var role string
		if err == nil {
			if role, err = s.spaces.Role(ctx, id, u.ID); err != nil {
				writeMessage(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		if role == "" {
			writeMessage(w, http.StatusNotFound, "Space not found.")
			return
		}
		d, err := s.store.Space(ctx, id)
		switch {
		case errors.Is(err, store.ErrUnavailable):
			writeMessage(w, http.StatusServiceUnavailable, "This space is temporarily unavailable.")
			return
		case err != nil:
			writeMessage(w, http.StatusNotFound, "Space not found.")
			return
		}
		ctx = context.WithValue(ctx, ctxSpace, s.newScope(id, role, d))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireWrite lets only space admins and editors change data; an API token
// is further limited by its own scope.
func (s *Server) requireWrite(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutating := r.Method != http.MethodGet && r.Method != http.MethodHead
		if mutating {
			if scope := sp(r); scope == nil || !spaces.CanWrite(scope.role) {
				writeMessage(w, http.StatusForbidden, "Read-only access")
				return
			}
			if t := apiTokenFrom(r); t != nil && !t.CanWrite() {
				writeMessage(w, http.StatusForbidden, "Read-only access")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) spacesIndex(w http.ResponseWriter, r *http.Request) {
	mine, err := s.spaces.ForUser(r.Context(), userFrom(r).ID)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]map[string]any, len(mine))
	for i, x := range mine {
		out[i] = map[string]any{"id": x.ID, "uuid": x.UUID, "name": x.Name, "role": x.Role}
	}
	writeData(w, http.StatusOK, out)
}
