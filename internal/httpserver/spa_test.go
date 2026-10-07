package httpserver

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestResolveAssetsUsesEmbedded(t *testing.T) {
	embedded := fstest.MapFS{"index.html": {Data: []byte("embedded")}}

	raw, err := fs.ReadFile(resolveAssets(embedded), "index.html")
	if err != nil || string(raw) != "embedded" {
		t.Fatalf("got %q, %v", raw, err)
	}
}

func TestResolveAssetsFallsBackToPublicDir(t *testing.T) {
	if got := resolveAssets(nil); got == nil {
		t.Fatal("no FS without an embedded frontend")
	}
}

func TestSPAServesEmbeddedFrontend(t *testing.T) {
	cfg, _ := testConfig(t)
	s := New(cfg, openTestStore(t, cfg), testKeys(t, cfg))
	s.assets = fstest.MapFS{
		"index.html":                {Data: []byte(`<html><body><div id="app"></div>{{range .JS}}<script src="{{.}}"></script>{{end}}</body></html>`)},
		"favicon.svg":               {Data: []byte("<svg/>")},
		"build/.vite/manifest.json": {Data: []byte(`{"resources/ts/main.tsx":{"file":"assets/main-abc.js","isEntry":true,"src":"resources/ts/main.tsx"}}`)},
		"build/assets/main-abc.js":  {Data: []byte("console.log(1)")},
	}
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)

	get := func(path string) (*http.Response, string) {
		t.Helper()
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		return res, string(raw)
	}

	if res, body := get("/favicon.svg"); res.StatusCode != http.StatusOK || body != "<svg/>" {
		t.Fatalf("favicon %d %q", res.StatusCode, body)
	}
	res, body := get("/build/assets/main-abc.js")
	if res.StatusCode != http.StatusOK || body != "console.log(1)" {
		t.Fatalf("asset %d %q", res.StatusCode, body)
	}
	if !strings.Contains(res.Header.Get("Cache-Control"), "immutable") {
		t.Fatalf("asset cache-control %q", res.Header.Get("Cache-Control"))
	}
	if res, body := get("/some/route"); res.StatusCode != http.StatusOK || !strings.Contains(body, `src="/build/assets/main-abc.js"`) {
		t.Fatalf("spa %d %q", res.StatusCode, body)
	}
	if res, _ := get("/build/assets/missing.js"); res.StatusCode != http.StatusNotFound {
		t.Fatalf("missing asset %d", res.StatusCode)
	}
}
