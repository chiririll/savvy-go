//go:build integration

package httpserver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
)

// dataRoots are the first path segments of the space data routes.
func dataRoots(s *Server) map[string]bool {
	r := chi.NewRouter()
	routing.SpaceWriteRoutes(r, s.deps())
	roots := map[string]bool{}
	_ = chi.Walk(r, func(_, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		roots[strings.SplitN(strings.TrimPrefix(route, "/"), "/", 2)[0]] = true
		return nil
	})
	return roots
}

// apiPath sends data requests written as /api/<data> to the shared test
// space, /api/spaces/{id}/<data>.
func (a *testApp) apiPath(path string) string {
	if a.space == nil {
		return path
	}
	return spacedPath(a.s, a.space.ID, path)
}

func spacedPath(s *Server, spaceID int64, path string) string {
	rest, ok := strings.CutPrefix(path, "/api/")
	if !ok {
		return path
	}
	root, _, _ := strings.Cut(rest, "/")
	root, _, _ = strings.Cut(root, "?")
	if dataRoots(s)[root] {
		return "/api/spaces/" + strconv.FormatInt(spaceID, 10) + "/" + rest
	}
	return path
}

func (a *testApp) request(method, path string, body any, token, csrf string) *http.Request {
	a.t.Helper()
	path = a.apiPath(path)
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			a.t.Fatal(err)
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, a.srv.URL+path, rdr)
	if err != nil {
		a.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "svy_session", Value: token})
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	return req
}

func (a *testApp) do(method, path string, body any, token, csrf string) *http.Response {
	a.t.Helper()
	res, err := a.client.Do(a.request(method, path, body, token, csrf))
	if err != nil {
		a.t.Fatal(err)
	}
	return res
}

func decodeJSON(t *testing.T, res *http.Response) map[string]any {
	t.Helper()
	defer res.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil && err != io.EOF {
		t.Fatal(err)
	}
	return out
}

// toString renders a decoded JSON value as text, for asserting that a response
// mentions some phrase. JSON escaping applies, so keep phrases free of quotes,
// angle brackets and ampersands.
func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if v == nil {
		return ""
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func cookieNamed(res *http.Response, name string) *http.Cookie {
	for _, c := range res.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}
