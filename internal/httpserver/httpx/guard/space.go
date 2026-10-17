package guard

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/auth"
	"savvy-go/internal/auth/permissions"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/refsync"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
)

// SpaceDeps is what the space guards need beyond the request itself.
type SpaceDeps struct {
	Spaces   spaces.Service
	Store    store.Store
	Refsync  refsync.Service
	NewScope func(id int64, role string, d store.DB) *httpx.SpaceScope
}

// WithSpace resolves the space named by {space} and the caller's role in
// it. Membership is checked before availability, so a space the caller does
// not belong to is a 404 whether or not it is broken (P42).
func (g *Guard) WithSpace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := httpx.User(r)
		if u == nil {
			httpx.WriteMessage(w, http.StatusUnauthorized, "Unauthenticated.")
			return
		}
		ctx := r.Context()
		id, err := strconv.ParseInt(chi.URLParam(r, "space"), 10, 64)
		var role string
		if err == nil {
			if role, err = g.Space.Spaces.Role(ctx, id, u.ID); err != nil {
				httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
				return
			}
			// A token reaches only the spaces it was given: for the rest the
			// space does not exist, as for a non-member.
			if t := httpx.Token(r); t != nil && !t.AllowsSpace(id) {
				role = ""
			}
		}
		if role == "" {
			httpx.WriteMessage(w, http.StatusNotFound, "Space not found.")
			return
		}
		db, err := g.Space.Store.Space(ctx, id)
		switch {
		case errors.Is(err, store.ErrUnavailable):
			httpx.WriteMessage(w, http.StatusServiceUnavailable, "This space is temporarily unavailable.")
			return
		case err != nil:
			httpx.WriteMessage(w, http.StatusNotFound, "Space not found.")
			return
		}
		ctx = httpx.WithScope(ctx, g.Space.NewScope(id, role, db))
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

// SpaceResource is the resource of a /spaces/{id}/... request path.
func SpaceResource(path string) (string, bool) {
	_, rest, ok := strings.Cut(path, "/spaces/")
	if !ok {
		return "", false
	}
	_, rest, _ = strings.Cut(rest, "/") // the space id
	seg, _, _ := strings.Cut(rest, "/")
	res, ok := spaceSegments[seg]
	return res, ok
}

// RequireTokenGrant holds an API token to the permissions it was given: read
// for a GET, write for anything that changes data. Sessions pass; for them the
// space role is the only limit.
func (g *Guard) RequireTokenGrant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if t := httpx.Token(r); t != nil {
			res, known := SpaceResource(r.URL.Path)
			level := permissions.LevelFor(r.Method)
			if !known || !t.Allows(res, level) {
				httpx.WriteMessage(w, http.StatusForbidden, "This API token has no "+permissions.New(res, level)+" permission.")
				return
			}
			if level != permissions.Read && !g.reachesSyncGroup(r, t, res) {
				httpx.WriteMessage(w, http.StatusForbidden, "This space syncs its "+res+" with other spaces: the API token must reach all of them to change them.")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// reachesSyncGroup is false when changing this resource also changes other
// spaces (a sync group mirrors categories and tags) that the token cannot
// reach. Without this, a token limited to one space could rewrite its
// neighbours. Whether the owner may edit each of them is checked by the
// change itself, as for a session.
func (g *Guard) reachesSyncGroup(r *http.Request, t *auth.APIToken, resource string) bool {
	var kind refsync.Kind
	switch resource {
	case "categories":
		kind = refsync.Categories
	case "tags":
		kind = refsync.Tags
	default:
		return true
	}
	grp, err := g.Space.Refsync.GroupOf(r.Context(), httpx.Scope(r).ID)
	if err != nil {
		return false
	}
	if grp == nil || !grp.Syncs(kind) {
		return true
	}
	for _, id := range grp.Members {
		if !t.AllowsSpace(id) {
			return false
		}
	}
	return true
}

// RequireWrite lets only space admins and editors change data. An API token
// has also passed RequireTokenGrant by now.
func (g *Guard) RequireWrite(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutating := r.Method != http.MethodGet && r.Method != http.MethodHead
		if mutating {
			if scope := httpx.Scope(r); scope == nil || !spaces.CanWrite(scope.Role) {
				httpx.WriteMessage(w, http.StatusForbidden, "Read-only access")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// RequireSpaceAdmin lets only admins of the request's space through.
func (g *Guard) RequireSpaceAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if scope := httpx.Scope(r); scope == nil || scope.Role != spaces.RoleAdmin {
			httpx.WriteMessage(w, http.StatusForbidden, "Only space administrators can do this.")
			return
		}
		next.ServeHTTP(w, r)
	})
}
