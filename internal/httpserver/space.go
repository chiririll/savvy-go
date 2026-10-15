package httpserver

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/auth"
	"savvy-go/internal/automation"
	"savvy-go/internal/importer"
	"savvy-go/internal/ledger"
	"savvy-go/internal/recurring"
	"savvy-go/internal/refsync"
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
			// A token reaches only the spaces it was given: for the rest the
			// space does not exist, as for a non-member.
			if t := apiTokenFrom(r); t != nil && !t.AllowsSpace(id) {
				role = ""
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

// spaceSegments maps the first path segment after /spaces/{id} to the
// resource a token needs for it. A segment missing here is closed to tokens.
var spaceSegments = map[string]string{
	"": "space", "members": "space", "settings": "space", "links": "space", "sync-group": "space",
	"accounts": "accounts", "accounts-balance-history": "accounts", "accounts-balance-comparison": "accounts",
	"automation-rules": "automation",
	"budgets":          "budgets",
	"categories":       "categories", "categories-summary": "categories",
	"currencies": "currencies",
	"debts":      "debts", "debts-summary": "debts",
	"recurring": "recurring", "recurring-upcoming": "recurring",
	"reports": "reports",
	"tags":    "tags",
	// Uploads exist for importing transactions.
	"transactions": "transactions", "transactions-summary": "transactions", "transactions-pending-summary": "transactions", "s3": "transactions",
	"transfers": "transfers",
}

// spaceResource is the resource of a /spaces/{id}/... request path.
func spaceResource(path string) (string, bool) {
	_, rest, ok := strings.Cut(path, "/spaces/")
	if !ok {
		return "", false
	}
	_, rest, _ = strings.Cut(rest, "/") // the space id
	seg, _, _ := strings.Cut(rest, "/")
	res, ok := spaceSegments[seg]
	return res, ok
}

// requireTokenGrant holds an API token to the permissions it was given: read
// for a GET, write for anything that changes data. Sessions pass; for them the
// space role is the only limit.
func (s *Server) requireTokenGrant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if t := apiTokenFrom(r); t != nil {
			res, known := spaceResource(r.URL.Path)
			level := auth.LevelFor(r.Method)
			if !known || !t.Allows(res, level) {
				writeMessage(w, http.StatusForbidden, "This API token has no "+auth.Perm(res, level)+" permission.")
				return
			}
			if level != auth.PermRead && !s.tokenReachesSyncGroup(r, t, res) {
				writeMessage(w, http.StatusForbidden, "This space syncs its "+res+" with other spaces: the API token must reach all of them to change them.")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// tokenReachesSyncGroup is false when changing this resource also changes other
// spaces (a sync group mirrors categories and tags) that the token cannot
// reach. Without this, a token limited to one space could rewrite its
// neighbours. Whether the owner may edit each of them is checked by the
// change itself, as for a session.
func (s *Server) tokenReachesSyncGroup(r *http.Request, t *auth.APIToken, resource string) bool {
	var kind refsync.Kind
	switch resource {
	case "categories":
		kind = refsync.Categories
	case "tags":
		kind = refsync.Tags
	default:
		return true
	}
	g, err := s.refsync.GroupOf(r.Context(), sp(r).id)
	if err != nil {
		return false
	}
	if g == nil || !g.Syncs(kind) {
		return true
	}
	for _, id := range g.Members {
		if !t.AllowsSpace(id) {
			return false
		}
	}
	return true
}

// requireWrite lets only space admins and editors change data. An API token
// has also passed requireTokenGrant by now.
func (s *Server) requireWrite(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutating := r.Method != http.MethodGet && r.Method != http.MethodHead
		if mutating {
			if scope := sp(r); scope == nil || !spaces.CanWrite(scope.role) {
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
	if t := apiTokenFrom(r); t != nil {
		mine = slices.DeleteFunc(mine, func(x spaces.Space) bool { return !t.AllowsSpace(x.ID) })
	}
	out := make([]map[string]any, len(mine))
	for i, x := range mine {
		out[i] = map[string]any{"id": x.ID, "uuid": x.UUID, "name": x.Name, "role": x.Role}
	}
	writeData(w, http.StatusOK, out)
}
