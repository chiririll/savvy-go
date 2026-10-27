//go:build integration

package httpserver

import (
	"net/http"
	"testing"

	"savvy-go/internal/auth"
)

// serverGet sends a request as is. a.do would send /api/currencies... to the
// shared test space, and these routes are server-wide.
func (a *testApp) serverGet(path, token string) *http.Response {
	a.t.Helper()
	req, err := http.NewRequest("GET", a.srv.URL+path, nil)
	if err != nil {
		a.t.Fatal(err)
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "svy_session", Value: token})
	}
	res, err := a.client.Do(req)
	if err != nil {
		a.t.Fatal(err)
	}
	return res
}

func dataList(t *testing.T, body map[string]any) []any {
	t.Helper()
	list, ok := body["data"].([]any)
	if !ok {
		t.Fatalf("data is not a list: %v", body)
	}
	return list
}

func TestServerCurrencyCatalogMatchesSpaceCatalog(t *testing.T) {
	a := newTestApp(t)
	user := a.issue(a.createUser("u@test.com", "secret1", roleEditor), false)

	res := a.serverGet("/api/catalog/currencies?base=USD", user.Token)
	if res.StatusCode != 200 {
		t.Fatalf("status %d", res.StatusCode)
	}
	list := dataList(t, decodeJSON(t, res))
	if len(list) == 0 {
		t.Fatal("empty catalog")
	}
	seen := map[string]bool{}
	for _, raw := range list {
		item := raw.(map[string]any)
		for _, key := range []string{"code", "name", "symbol", "decimals", "rate"} {
			if _, ok := item[key]; !ok {
				t.Fatalf("item %v lacks %q", item, key)
			}
		}
		seen[item["code"].(string)] = true
	}
	if !seen["USD"] || !seen["EUR"] {
		t.Fatalf("USD and EUR must be offered: %v", seen)
	}
	// The space catalog hides what the space owns; the server one never does.
	sess := a.issue(a.createUser("owner@test.com", "secret1", auth.RoleAdmin), false)
	space := a.do("POST", "/api/spaces", map[string]any{"name": "S"}, sess.Token, sess.CSRF)
	if space.StatusCode != 201 {
		t.Fatalf("create space %d", space.StatusCode)
	}
	id := int64(decodeJSON(t, space)["data"].(map[string]any)["id"].(float64))
	if cur := a.inSpace(id, "POST", "/api/currencies", map[string]any{"code": "USD", "name": "US Dollar", "symbol": "$"},
		sess); cur.StatusCode != 201 {
		t.Fatalf("add currency %d", cur.StatusCode)
	}
	own := dataList(t, decodeJSON(t, a.inSpace(id, "GET", "/api/currencies/catalog", nil, sess)))
	if len(own) != len(list)-1 {
		t.Fatalf("space catalog %d, server catalog %d: only the owned currency may differ", len(own), len(list))
	}
}

func TestServerCurrencyCatalogRejectsBadBase(t *testing.T) {
	a := newTestApp(t)
	user := a.issue(a.createUser("u@test.com", "secret1", roleEditor), false)
	res := a.serverGet("/api/catalog/currencies?base=../x", user.Token)
	if res.StatusCode != 422 {
		t.Fatalf("bad base got %d", res.StatusCode)
	}
	res.Body.Close()
}

func TestPresetsServeTheCategoriesTags(t *testing.T) {
	a := newTestApp(t)
	user := a.issue(a.createUser("u@test.com", "secret1", roleViewer), false)

	res := a.serverGet("/api/catalog/presets", user.Token)
	if res.StatusCode != 200 {
		t.Fatalf("status %d", res.StatusCode)
	}
	data := decodeJSON(t, res)["data"].(map[string]any)
	cats := data["categories"].([]any)
	if len(cats) == 0 {
		t.Fatal("no preset categories")
	}
	defaults := map[string]string{}
	for _, raw := range cats {
		c := raw.(map[string]any)
		for _, key := range []string{"name", "type", "icon", "color"} {
			if s, _ := c[key].(string); s == "" {
				t.Fatalf("category %v lacks %q", c, key)
			}
		}
		if c["isDefault"] == true {
			defaults[c["type"].(string)] = c["name"].(string)
		}
	}
	if defaults["expense"] != "#OTHER" || defaults["income"] != "#OTHER_INCOME" || len(defaults) != 2 {
		t.Fatalf("defaults %v", defaults)
	}
	tags := data["tags"].([]any)
	if len(tags) == 0 {
		t.Fatal("no preset tags")
	}
}

// The catalogs of the setup screens are for people who are signed in.
func TestServerCatalogsNeedSignedInUser(t *testing.T) {
	a := newTestApp(t)
	for _, path := range []string{"/api/catalog/currencies", "/api/catalog/presets"} {
		status(t, a.serverGet(path, ""), 401, path)
	}
}

func TestServerCurrencyCatalogFlagsBuiltInList(t *testing.T) {
	a := newTestApp(t) // its rate source is offline: only the built-in list is known
	user := a.issue(a.createUser("u@test.com", "secret1", roleEditor), false)

	res := a.serverGet("/api/catalog/currencies?base=USD", user.Token)
	body := decodeJSON(t, res)
	meta, _ := body["meta"].(map[string]any)
	if meta["source"] != "fallback" {
		t.Fatalf("meta %v, want source=fallback", body["meta"])
	}
}
