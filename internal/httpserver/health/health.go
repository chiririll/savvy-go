package health

import (
	"net/http"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/platform/version"
	"savvy-go/internal/store"
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

// Handler serves the liveness and readiness probes.
type Handler struct{ Store store.Store }

func (h Handler) Live(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteHealth(w, http.StatusOK, healthReport{
		Status:    "pass",
		ReleaseID: version.Value,
	})
}

// readyz passes once the store has opened and migrated every database. A
// space that failed to migrate does not fail the probe (the other spaces
// work); it is listed as a warning.
func (h Handler) Ready(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC().Format(time.RFC3339)
	checks := map[string][]healthChk{}
	passing := true

	conn := healthChk{ComponentType: "datastore", Status: "pass", Time: now}
	if err := db.Ping(r.Context(), h.Store.Server()); err != nil {
		passing = false
		conn.Status = "fail"
		conn.Output = err.Error()
	}
	checks["store:connectivity"] = []healthChk{conn}

	st := h.Store.Status()
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
	httpx.WriteHealth(w, status, healthReport{
		Status:    reportStatus,
		ReleaseID: version.Value,
		Checks:    checks,
	})
}
