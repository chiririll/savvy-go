package httpserver

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/migrate"
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

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC().Format(time.RFC3339)
	checks := map[string][]healthChk{}
	passing := true

	conn := healthChk{ComponentType: "datastore", Status: "pass", Time: now}
	if err := db.Ready(r.Context(), s.db); err != nil {
		passing = false
		conn.Status = "fail"
		conn.Output = err.Error()
	}
	checks["sqlite:connectivity"] = []healthChk{conn}

	mig := healthChk{ComponentType: "component", Status: "pass", Time: now, ObservedUnit: "pending"}
	pending, err := migrate.PendingCount(r.Context(), s.db)
	if err != nil {
		passing = false
		mig.Status = "fail"
		mig.Output = err.Error()
	} else {
		value := pending
		if pending < 0 {
			value = pending
			passing = false
			mig.Status = "fail"
		} else if pending != 0 {
			passing = false
			mig.Status = "fail"
		}
		mig.ObservedValue = &value
		if pending < 0 {
			mig.ObservedValue = intPtr(pending)
		}
	}
	checks["schema:migrations"] = []healthChk{mig}

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

func intPtr(n int) *int { return &n }

// ReadyForTraffic is used by the process itself (startup / probes).
func ReadyForTraffic(ctx context.Context, sqlDB *sql.DB) bool {
	if err := db.Ready(ctx, sqlDB); err != nil {
		return false
	}
	pending, err := migrate.PendingCount(ctx, sqlDB)
	return err == nil && pending == 0
}
