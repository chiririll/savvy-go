package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/ledger"
)

// refreshable is a rate source that can say it was asked to retry.
type refreshable struct {
	refreshes int
	builtIn   bool
	// recoverOnRefresh turns the built-in list into the full one on Refresh.
	recoverOnRefresh bool
}

func (r *refreshable) Catalog(context.Context, string) []ledger.Currency { return nil }
func (r *refreshable) Rates(context.Context, string) (map[string]decimal.Decimal, error) {
	return nil, nil
}
func (r *refreshable) CatalogStatus(context.Context, string) ([]ledger.Currency, bool) {
	return []ledger.Currency{{Code: "USD", Name: "US Dollar", Decimals: 2}}, r.builtIn
}
func (r *refreshable) Refresh() bool {
	r.refreshes++
	if r.recoverOnRefresh {
		r.builtIn = false
	}
	return true
}

func get(h Handler, query string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.currencies(w, httptest.NewRequest(http.MethodGet, "/currencies/catalog"+query, nil))
	return w
}

func TestCatalogRefreshParamAsksSourceToRetry(t *testing.T) {
	src := &refreshable{builtIn: true, recoverOnRefresh: true}
	h := Handler{Rates: src}

	w := get(h, "")
	if src.refreshes != 0 {
		t.Fatalf("a plain request refreshed %d times", src.refreshes)
	}
	if !strings.Contains(w.Body.String(), `"source":"fallback"`) {
		t.Fatalf("body %s, want fallback", w.Body)
	}

	w = get(h, "?refresh=1")
	if src.refreshes != 1 {
		t.Fatalf("refreshes=%d, want 1", src.refreshes)
	}
	if !strings.Contains(w.Body.String(), `"source":"live"`) {
		t.Fatalf("body %s, want live after the retry", w.Body)
	}
}

func TestCatalogRefreshOtherValuesIgnored(t *testing.T) {
	src := &refreshable{}
	for _, q := range []string{"?refresh=0", "?refresh=", "?refresh=no"} {
		get(Handler{Rates: src}, q)
	}
	if src.refreshes != 0 {
		t.Fatalf("refreshes=%d, want 0", src.refreshes)
	}
}

func TestCatalogRefreshWithSourceThatCannotRefresh(t *testing.T) {
	w := get(Handler{Rates: stubPlain{}}, "?refresh=1")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
}

type stubPlain struct{}

func (stubPlain) Catalog(context.Context, string) []ledger.Currency { return nil }
func (stubPlain) Rates(context.Context, string) (map[string]decimal.Decimal, error) {
	return nil, nil
}
