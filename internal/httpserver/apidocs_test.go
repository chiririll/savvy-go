package httpserver

import (
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

var (
	specPathRe   = regexp.MustCompile(`^  (/[^\s:]*):\s*$`)
	specMethodRe = regexp.MustCompile(`^    (get|post|put|patch|delete):\s*$`)
	paramRe      = regexp.MustCompile(`\{[^}]+\}`)
)

// specOperations returns "METHOD /path" for every operation in openapi.yaml, with path
// parameter names normalised so {id} and {upload} compare equal.
func specOperations(t *testing.T) map[string]bool {
	t.Helper()
	spec, err := os.ReadFile("../../public/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	ops := map[string]bool{}
	path := ""
	inPaths := false
	for _, line := range strings.Split(string(spec), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "paths:") {
			inPaths = true
			continue
		}
		if inPaths && line != "" && !strings.HasPrefix(line, " ") {
			break
		}
		if m := specPathRe.FindStringSubmatch(line); m != nil {
			path = paramRe.ReplaceAllString(m[1], "{}")
			continue
		}
		if m := specMethodRe.FindStringSubmatch(line); m != nil && path != "" {
			ops[strings.ToUpper(m[1])+" /api"+path] = true
		}
	}
	return ops
}

// undocumented lists routes that are intentionally absent from the spec: the SPA's login and
// SSO plumbing, uploads, imports and backups (session-only), and admin-only management.
var undocumented = []string{
	"/api/docs", "/api/openapi.yaml",
	"/api/auth/status", "/api/auth/me", "/api/auth/register", "/api/auth/bootstrap", "/api/auth/login",
	"/api/auth/password", "/api/auth/logout", "/api/auth/logout-others", "/api/auth/verify-email",
	"/api/auth/2fa", "/api/auth/sso", "/api/auth/webauthn",
	"/api/uploads", "/api/s3/multipart", "/api/transactions/import",
	"/api/oauth", "/api/backups", "/api/identity-providers", "/api/spaces/import",
	"/api/admin", "/api/monitoring", "/api/invitations", "/api/invitation-methods",
}

// undocumentedInSpace lists routes under /api/spaces/{space} that are
// absent for the same reasons: uploads, imports, backups, invitations and the
// audit log are session-only.
var undocumentedInSpace = []string{"/s3/multipart", "/transactions/import", "/backups", "/invitations", "/audit"}

func isUndocumented(path string) bool {
	if rest, ok := strings.CutPrefix(path, "/api/spaces/{space}"); ok {
		for _, p := range undocumentedInSpace {
			if strings.HasPrefix(rest, p) {
				return true
			}
		}
	}
	for _, p := range undocumented {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}

func TestOpenAPICoversRoutes(t *testing.T) {
	a := newTestApp(t)
	spec := specOperations(t)
	if len(spec) < 50 {
		t.Fatalf("spec parsed only %d operations", len(spec))
	}

	routes := map[string]bool{}
	err := chi.Walk(a.s.mux, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		route = strings.TrimSuffix(route, "/")
		if !strings.HasPrefix(route, "/api/") || isUndocumented(route) {
			return nil
		}
		// Admin-only user management is documented only for its read endpoints.
		if method != "GET" && strings.HasPrefix(route, "/api/users") {
			return nil
		}
		routes[method+" "+paramRe.ReplaceAllString(route, "{}")] = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for op := range routes {
		if !spec[op] {
			t.Errorf("route %s is missing from public/openapi.yaml", op)
		}
	}
	for op := range spec {
		if !routes[op] {
			t.Errorf("openapi.yaml documents %s, which is not a route", op)
		}
	}
}

// The reference and its spec are static files of the frontend; the documented
// /api URLs lead to them.
func TestAPIDocsPage(t *testing.T) {
	a := newTestApp(t)
	a.client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	for from, to := range map[string]string{"/api/docs": "/api-docs.html", "/api/openapi.yaml": "/openapi.yaml"} {
		res := a.do("GET", from, nil, "", "")
		res.Body.Close()
		if res.StatusCode != http.StatusFound || res.Header.Get("Location") != to {
			t.Errorf("%s: %d to %q, want %s", from, res.StatusCode, res.Header.Get("Location"), to)
		}
	}
	page, err := os.ReadFile("../../public/api-docs.html")
	if err != nil || !strings.Contains(string(page), "swagger-ui") || !strings.Contains(string(page), `"/openapi.yaml"`) {
		t.Fatalf("public/api-docs.html: %v", err)
	}
	spec, err := os.ReadFile("../../public/openapi.yaml")
	if err != nil || !strings.Contains(string(spec), "openapi: 3.0.3") {
		t.Fatalf("public/openapi.yaml: %v", err)
	}
}
