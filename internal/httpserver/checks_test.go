package httpserver

import (
	"testing"

	"savvy-go/internal/auth"
)

func TestChecksEndpointIsForServerAdmins(t *testing.T) {
	a := newTestApp(t)
	editor := a.issue(a.createUser("e@test.com", "secret1", roleEditor), false)
	admin := a.issue(a.createUser("a@test.com", "secret1", auth.RoleAdmin), false)

	status(t, a.do("GET", "/api/admin/checks", nil, "", ""), 401, "anonymous")
	status(t, a.do("GET", "/api/admin/checks", nil, editor.Token, ""), 403, "plain user")

	res := a.do("GET", "/api/admin/checks", nil, admin.Token, "")
	body := decodeJSON(t, res)
	if res.StatusCode != 200 {
		t.Fatalf("admin %d %v", res.StatusCode, body)
	}
	list, ok := body["data"].([]any)
	if !ok {
		t.Fatalf("data is not a list: %v", body)
	}
	ids := map[string]bool{}
	for _, item := range list {
		ids[item.(map[string]any)["id"].(string)] = true
	}
	// The test server has no mail and no backup yet.
	if !ids["mail_off"] || !ids["backup_missing"] {
		t.Fatalf("expected mail_off and backup_missing, got %v", ids)
	}
	if ids["app_url_missing"] {
		t.Fatalf("test server has a public URL: %v", ids)
	}
}
