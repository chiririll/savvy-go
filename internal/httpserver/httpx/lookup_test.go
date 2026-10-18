package httpx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type item struct{ ID int64 }

func lookupRequest(id string) *http.Request {
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", id)
	r := httptest.NewRequest("GET", "/x/"+id, nil)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rc))
}

func TestLookup(t *testing.T) {
	found := func(_ context.Context, id int64) (*item, error) { return &item{ID: id}, nil }
	missing := func(context.Context, int64) (*item, error) { return nil, nil }
	broken := func(context.Context, int64) (*item, error) { return nil, errors.New("db down") }

	cases := []struct {
		name   string
		id     string
		byID   func(context.Context, int64) (*item, error)
		status int
		want   bool
	}{
		{"found", "7", found, 200, true},
		{"missing", "7", missing, 404, false},
		{"bad id", "abc", missing, 404, false},
		{"repo error is not a 404", "7", broken, 500, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			got := Lookup(w, lookupRequest(tc.id), tc.byID)
			if (got != nil) != tc.want {
				t.Fatalf("got %v, want found=%v", got, tc.want)
			}
			if got != nil && got.ID != 7 {
				t.Errorf("id = %d, want 7", got.ID)
			}
			if got == nil && w.Code != tc.status {
				t.Errorf("status = %d, want %d", w.Code, tc.status)
			}
		})
	}
}
