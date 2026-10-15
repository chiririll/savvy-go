package httpserver

import (
	"net/http"
	"sort"
	"time"
)

// Server checks tell an admin what is misconfigured or running out. They are
// computed from the current state on every request, never stored: a fixed
// problem disappears by itself and nothing can go stale. The ids are stable;
// the SPA owns the wording and the link to the page that fixes each one.

const (
	severityError   = "error"
	severityWarning = "warning"
	severityInfo    = "info"

	// Server backups are made by hand, so "stale" is generous.
	backupStaleAfter = 30 * 24 * time.Hour

	diskWarnPercent  = 75.0
	diskErrorPercent = 90.0
)

type serverCheck struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	// Value is a number the message shows (disk percent, days since a backup);
	// null when the check has none.
	Value any `json:"value"`
}

// checkInput is what the checks look at, so they can be tested without a server.
type checkInput struct {
	AppURL      string
	MailEnabled bool
	MailProblem string
	Writable    bool       // the config file can be edited from the panel
	LastBackup  *time.Time // newest valid server backup, nil if there is none
	DiskPercent *float64   // used share of the data volume, nil if unknown
}

func serverChecks(in checkInput, now time.Time) []serverCheck {
	out := []serverCheck{}
	add := func(id, severity string, value any) {
		out = append(out, serverCheck{ID: id, Severity: severity, Value: value})
	}

	if in.AppURL == "" {
		add("app_url_missing", severityWarning, nil)
	}
	switch {
	case in.MailProblem != "":
		add("mail_invalid", severityError, nil)
	case !in.MailEnabled:
		add("mail_off", severityWarning, nil)
	}
	switch {
	case in.LastBackup == nil:
		add("backup_missing", severityWarning, nil)
	case now.Sub(*in.LastBackup) > backupStaleAfter:
		add("backup_stale", severityWarning, int(now.Sub(*in.LastBackup).Hours()/24))
	}
	if in.DiskPercent != nil {
		switch p := *in.DiskPercent; {
		case p >= diskErrorPercent:
			add("disk_filling", severityError, p)
		case p >= diskWarnPercent:
			add("disk_filling", severityWarning, p)
		}
	}
	if !in.Writable {
		add("config_read_only", severityInfo, nil)
	}

	rank := map[string]int{severityError: 0, severityWarning: 1, severityInfo: 2}
	sort.SliceStable(out, func(i, j int) bool { return rank[out[i].Severity] < rank[out[j].Severity] })
	return out
}

// checksIndex lists the problems an admin should know about (server admins, by
// route). A source that cannot be read counts as "unknown", not as a problem.
func (s *Server) checksIndex(w http.ResponseWriter, r *http.Request) {
	mail, problem := s.mailSettings()
	in := checkInput{
		AppURL:      s.cfg.URL(),
		MailEnabled: s.mailer().Enabled() && mail.Enabled,
		MailProblem: problem,
		Writable:    s.cfg.Live.Writable(),
		DiskPercent: s.diskPercent(),
	}
	if list, err := s.backups.ServerBackups(); err == nil {
		for _, b := range list {
			if b.Valid && b.CreatedAt != nil && (in.LastBackup == nil || b.CreatedAt.After(*in.LastBackup)) {
				in.LastBackup = b.CreatedAt
			}
		}
	} else {
		// Unknown: do not claim there is no backup.
		in.LastBackup = addr(time.Now())
	}
	writeData(w, http.StatusOK, serverChecks(in, time.Now()))
}

// diskPercent is the used share of the volume that holds the uploads, nil when
// the platform cannot tell.
func (s *Server) diskPercent() *float64 {
	total, free, ok := volumeSpace(s.cfg.UploadsDir)
	if !ok || total <= 0 {
		return nil
	}
	p := float64(total-free) / float64(total) * 100
	return &p
}

func addr[T any](v T) *T { return &v }
