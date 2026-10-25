//go:build integration

package httpserver

import (
	"net/http"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/auth"
)

var routeParam = regexp.MustCompile(`\{[^}]+\}`)

// adminRoutes lists "METHOD /path" for every route under /api/admin, path parameters filled in.
func adminRoutes(t *testing.T, s *Server) []string {
	t.Helper()
	var out []string
	err := chi.Walk(s.mux, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		route = strings.TrimSuffix(route, "/")
		if strings.HasPrefix(route, "/api/admin/") {
			out = append(out, method+" "+routeParam.ReplaceAllString(route, "1"))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)
	return out
}

// Every route under /api/admin is for server admins, whatever its handler does: the tier is
// what guards it, so a new admin route cannot be left open by forgetting a check.
func TestAdminRoutesAreForServerAdmins(t *testing.T) {
	a := newTestApp(t)
	user := a.issue(a.createUser("user@test.com", "secret1", roleEditor), false)
	admin := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)

	routes := adminRoutes(t, a.s)
	if len(routes) < 8 {
		t.Fatalf("found only %d admin routes: %v", len(routes), routes)
	}
	for _, route := range routes {
		method, path, _ := strings.Cut(route, " ")
		as := func(sess *auth.Issued) int {
			var res *http.Response
			if sess == nil {
				res = a.do(method, path, map[string]any{}, "", "")
			} else {
				res = a.do(method, path, map[string]any{}, sess.Token, sess.CSRF)
			}
			res.Body.Close()
			return res.StatusCode
		}
		if got := as(nil); got != http.StatusUnauthorized {
			t.Errorf("%s signed out = %d, want 401", route, got)
		}
		if got := as(user); got != http.StatusForbidden {
			t.Errorf("%s as a plain user = %d, want 403", route, got)
		}
		// Past the gate the handler decides; it may refuse the made-up ids, never the role.
		if got := as(admin); got == http.StatusUnauthorized || got == http.StatusForbidden {
			t.Errorf("%s as an admin = %d", route, got)
		}
	}
}
