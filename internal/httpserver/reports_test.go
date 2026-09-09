package httpserver

import (
	"testing"

	"github.com/chiririll/savvy-plus/internal/auth"
)

func TestReportsOverview(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("r@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)
	res := a.do("GET", "/api/reports/overview", nil, sess.Token, "")
	body := decodeJSON(t, res)
	if res.StatusCode != 200 {
		t.Fatalf("overview %d %v", res.StatusCode, body)
	}
	if body["income"] == nil || body["expenses"] == nil {
		t.Fatalf("shape %v", body)
	}
}
