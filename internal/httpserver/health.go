package httpserver

import (
	"net/http"
	"time"

	"savvy-go/internal/version"
)

type healthReport struct {
	Status    string                 `json:"status"`
	ReleaseID string                 `json:"releaseId"`
	Checks    map[string][]healthChk `json:"checks,omitempty"`
}

type healthChk struct {
	ComponentType string `json:"componentType"`
	Status        string `json:"status"`
	Time          string `json:"time"`
	ObservedValue *int   `json:"observedValue,omitempty"`
	ObservedUnit  string `json:"observedUnit,omitempty"`
	Output        string `json:"output,omitempty"`
}

func (s *Server) livez(w http.ResponseWriter, _ *http.Request) {
	writeHealth(w, http.StatusOK, healthReport{
		Status:    "pass",
		ReleaseID: version.Value,
	})
}

// readyz passes once the store has opened and migrated every database. A
// space that failed to migrate does not fail the probe (the other spaces
// work); it is listed as a warning.
func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC().Format(time.RFC3339)
	checks := map[string][]healthChk{}
	passing := true

	conn := healthChk{ComponentType: "datastore", Status: "pass", Time: now}
	var one int
	if err := s.store.Server().QueryRowContext(r.Context(), "SELECT 1").Scan(&one); err != nil {
		passing = false
		conn.Status = "fail"
		conn.Output = err.Error()
	}
	checks["store:connectivity"] = []healthChk{conn}

	st := s.store.Status()
	mig := healthChk{ComponentType: "component", Status: "pass", Time: now, ObservedUnit: "unavailable spaces"}
	if !st.Ready {
		passing = false
		mig.Status = "fail"
		mig.Output = "databases are still being opened and migrated"
	}
	unavailable := len(st.Unavailable)
	mig.ObservedValue = &unavailable
	if unavailable > 0 && st.Ready {
		mig.Status = "warn"
	}
	checks["store:migrations"] = []healthChk{mig}

	status := http.StatusOK
	reportStatus := "pass"
	if !passing {
		status = http.StatusServiceUnavailable
		reportStatus = "fail"
	}
	writeHealth(w, status, healthReport{
		Status:    reportStatus,
		ReleaseID: version.Value,
		Checks:    checks,
	})
}
