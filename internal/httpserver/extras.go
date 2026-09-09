package httpserver

import (
	"encoding/json"
	"net/http"
)

func (s *Server) emptyList(w http.ResponseWriter, _ *http.Request) {
	writeData(w, http.StatusOK, []any{})
}

func (s *Server) emptyCreated(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	writeJSON(w, http.StatusNotImplemented, map[string]string{
		"message": "This endpoint is not implemented in the Go backend yet.",
	})
}

func (s *Server) recurringUpcoming(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"data": []any{}})
}

func (s *Server) automationTriggers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"data": []map[string]string{
			{"value": "on_transaction_create", "label": "On transaction create"},
			{"value": "on_transaction_update", "label": "On transaction update"},
		},
	})
}

func (s *Server) ssoProviders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"data": []any{}})
}

func (s *Server) ssoPresets(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"data": []string{
		"github", "google", "oidc", "saml", "keycloak", "authentik", "okta", "entra", "gitlab",
	}})
}
