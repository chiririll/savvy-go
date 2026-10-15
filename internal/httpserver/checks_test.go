package httpserver

import (
	"testing"
	"time"

	"savvy-go/internal/auth"
)

func checkIDs(list []serverCheck) map[string]string {
	out := map[string]string{}
	for _, c := range list {
		out[c.ID] = c.Severity
	}
	return out
}

func TestServerChecks(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	healthy := checkInput{
		AppURL:      "https://savvy.example",
		MailEnabled: true,
		Writable:    true,
		LastBackup:  addr(now.Add(-24 * time.Hour)),
		DiskPercent: addr(40.0),
	}

	tests := []struct {
		name string
		edit func(*checkInput)
		want map[string]string
	}{
		{"healthy server has no checks", func(*checkInput) {}, map[string]string{}},
		{"no public url", func(in *checkInput) { in.AppURL = "" }, map[string]string{"app_url_missing": "warning"}},
		{"mail off", func(in *checkInput) { in.MailEnabled = false }, map[string]string{"mail_off": "warning"}},
		{"mail section invalid", func(in *checkInput) { in.MailEnabled, in.MailProblem = false, "host is required" }, map[string]string{"mail_invalid": "error"}},
		{"never backed up", func(in *checkInput) { in.LastBackup = nil }, map[string]string{"backup_missing": "warning"}},
		{"backup 29 days old is fine", func(in *checkInput) { in.LastBackup = addr(now.Add(-29 * 24 * time.Hour)) }, map[string]string{}},
		{"backup 31 days old", func(in *checkInput) { in.LastBackup = addr(now.Add(-31 * 24 * time.Hour)) }, map[string]string{"backup_stale": "warning"}},
		{"config read-only", func(in *checkInput) { in.Writable = false }, map[string]string{"config_read_only": "info"}},
		{"disk 75% is filling", func(in *checkInput) { in.DiskPercent = addr(75.0) }, map[string]string{"disk_filling": "warning"}},
		{"disk 90% is critical", func(in *checkInput) { in.DiskPercent = addr(90.0) }, map[string]string{"disk_filling": "error"}},
		{"unknown disk is not a warning", func(in *checkInput) { in.DiskPercent = nil }, map[string]string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := healthy
			tc.edit(&in)
			got := checkIDs(serverChecks(in, now))
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for id, sev := range tc.want {
				if got[id] != sev {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestServerChecksSortedBySeverity(t *testing.T) {
	now := time.Now()
	got := serverChecks(checkInput{MailProblem: "bad", Writable: false, DiskPercent: addr(50.0)}, now)
	var order []string
	for _, c := range got {
		order = append(order, c.Severity)
	}
	rank := map[string]int{"error": 0, "warning": 1, "info": 2}
	for i := 1; i < len(order); i++ {
		if rank[order[i-1]] > rank[order[i]] {
			t.Fatalf("not sorted by severity: %v", order)
		}
	}
}

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
