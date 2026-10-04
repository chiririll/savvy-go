package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"savvy-go/internal/domain"
	"savvy-go/internal/settings"
	"savvy-go/internal/store"
)

const ctxSpace ctxKey = 100

// spaceScope is the space a request works in: the caller's role there and
// the domain services bound to its database.
type spaceScope struct {
	id         int64
	role       string
	db         store.DB
	settings   settings.Store
	currencies domain.Currencies
	accounts   domain.Accounts
	categories domain.Categories
	tags       domain.Tags
	txs        domain.Transactions
	debts      domain.Debts
	recurring  domain.RecurringStore
	budgets    domain.Budgets
	automation domain.Automation
	reports    domain.Reports
	imports    domain.Imports
}

func (s *Server) newScope(id int64, role string, d store.DB) *spaceScope {
	txs := domain.Transactions{DB: d}
	accounts := domain.Accounts{DB: d}
	return &spaceScope{
		id:         id,
		role:       role,
		db:         d,
		settings:   settings.Store{DB: d, Space: true},
		currencies: domain.Currencies{DB: d},
		accounts:   accounts,
		categories: domain.Categories{DB: d},
		tags:       domain.Tags{DB: d},
		txs:        txs,
		debts:      domain.Debts{Accounts: accounts, Transactions: txs},
		recurring:  domain.RecurringStore{DB: d, Txs: txs},
		budgets:    domain.Budgets{DB: d},
		automation: domain.Automation{DB: d, Txs: txs},
		reports:    domain.Reports{DB: d, Loc: s.cfg.Location},
		imports:    domain.Imports{DB: d, Uploads: s.uploads, Txs: txs},
	}
}

// sp is the space of a request that passed withSpace.
func sp(r *http.Request) *spaceScope {
	scope, _ := r.Context().Value(ctxSpace).(*spaceScope)
	return scope
}

// spaceHeader names the space a request works in until the routes move under
// /api/spaces/{space}; without it the caller's first space is used.
const spaceHeader = "X-Space-ID"

// withSpace resolves the space of the request and the caller's role in it.
// Membership is checked before availability, so a space the caller does not
// belong to is a 404 whether or not it is broken.
func (s *Server) withSpace(next http.Handler) http.Handler { return s.resolveSpace(next, true) }

// withOptionalSpace resolves the space like withSpace but lets a caller who
// belongs to no space through without one (sp(r) is then nil).
func (s *Server) withOptionalSpace(next http.Handler) http.Handler {
	return s.resolveSpace(next, false)
}

func (s *Server) resolveSpace(next http.Handler, required bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFrom(r)
		if u == nil {
			writeMessage(w, http.StatusUnauthorized, "Unauthenticated.")
			return
		}
		ctx := r.Context()
		var id int64
		var role string
		raw := strings.TrimSpace(r.Header.Get(spaceHeader))
		if raw != "" {
			parsed, err := strconv.ParseInt(raw, 10, 64)
			if err == nil {
				id = parsed
				role, err = s.spaces.Role(ctx, id, u.ID)
			}
			if err != nil {
				writeMessage(w, http.StatusInternalServerError, err.Error())
				return
			}
		} else {
			mine, err := s.spaces.ForUser(ctx, u.ID)
			if err != nil {
				writeMessage(w, http.StatusInternalServerError, err.Error())
				return
			}
			if len(mine) > 0 {
				id, role = mine[0].ID, mine[0].Role
			}
		}
		if role == "" {
			if !required && raw == "" {
				next.ServeHTTP(w, r)
				return
			}
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
			if scope := sp(r); scope == nil || !domain.CanWriteSpace(scope.role) {
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
