//go:build integration

package httpserver

import (
	"testing"

	"savvy-go/internal/auth"
)

func TestMonitoringIsForServerAdmins(t *testing.T) {
	a := newTestApp(t)
	user := a.issue(a.createUser("user@test.com", "secret1", roleEditor), false)
	admin := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)

	for path, field := range map[string]string{"/api/monitoring/storage": "volume", "/api/monitoring/resources": "cpu"} {
		status(t, a.do("GET", path, nil, "", ""), 401, path+" signed out")
		status(t, a.do("GET", path, nil, user.Token, ""), 403, path+" plain user")
		body := decodeJSON(t, a.do("GET", path, nil, admin.Token, ""))
		if body[field] == nil {
			t.Errorf("%s as an admin lacks %q: %v", path, field, body)
		}
	}
}
