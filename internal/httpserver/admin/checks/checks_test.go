package checks

import (
	"testing"
	"time"
)

func checkIDs(list []Check) map[string]string {
	out := map[string]string{}
	for _, c := range list {
		out[c.ID] = c.Severity
	}
	return out
}

func TestRun(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	healthy := Input{
		AppURL:      "https://savvy.example",
		MailEnabled: true,
		Writable:    true,
		LastBackup:  addr(now.Add(-24 * time.Hour)),
		DiskPercent: addr(40.0),
	}

	tests := []struct {
		name string
		edit func(*Input)
		want map[string]string
	}{
		{"healthy server has no checks", func(*Input) {}, map[string]string{}},
		{"no public url", func(in *Input) { in.AppURL = "" }, map[string]string{"app_url_missing": "warning"}},
		{"mail off", func(in *Input) { in.MailEnabled = false }, map[string]string{"mail_off": "info"}},
		{
			"mail section invalid",
			func(in *Input) { in.MailEnabled, in.MailProblem = false, "host is required" },
			map[string]string{"mail_invalid": "error"},
		},
		{"never backed up", func(in *Input) { in.LastBackup = nil }, map[string]string{"backup_missing": "info"}},
		{
			"backup 29 days old is fine",
			func(in *Input) { in.LastBackup = addr(now.Add(-29 * 24 * time.Hour)) },
			map[string]string{},
		},
		{
			"backup 31 days old",
			func(in *Input) { in.LastBackup = addr(now.Add(-31 * 24 * time.Hour)) },
			map[string]string{"backup_stale": "warning"},
		},
		{"config read-only", func(in *Input) { in.Writable = false }, map[string]string{"config_read_only": "info"}},
		{
			"disk 75% is filling",
			func(in *Input) { in.DiskPercent = addr(75.0) },
			map[string]string{"disk_filling": "warning"},
		},
		{"disk 90% is critical", func(in *Input) { in.DiskPercent = addr(90.0) }, map[string]string{"disk_filling": "error"}},
		{"unknown disk is not a warning", func(in *Input) { in.DiskPercent = nil }, map[string]string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := healthy
			tc.edit(&in)
			got := checkIDs(Run(in, now))
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
	got := Run(Input{MailProblem: "bad", Writable: false, DiskPercent: addr(50.0)}, now)
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
