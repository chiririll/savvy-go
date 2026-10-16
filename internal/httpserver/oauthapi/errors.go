package oauthapi

import (
	"errors"
	"net/http"

	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/oauth"
)

// writeOAuthError sends an RFC 6749 §5.2 error body.
func writeOAuthError(w http.ResponseWriter, err error) {
	api.OpenCORS(w)
	w.Header().Set("Cache-Control", "no-store")
	var oe *oauth.Error
	if !errors.As(err, &oe) {
		oe = &oauth.Error{Code: "server_error", Description: err.Error()}
	}
	status := http.StatusBadRequest
	switch oe.Code {
	case "invalid_client":
		status = http.StatusUnauthorized
	case "access_denied":
		status = http.StatusForbidden
	case "server_error":
		status = http.StatusInternalServerError
	}
	api.WriteJSON(w, status, map[string]string{"error": oe.Code, "error_description": oe.Description})
}

// writeOAuthAPIError is writeOAuthError for the session API: the SPA reads message.
func writeOAuthAPIError(w http.ResponseWriter, err error) {
	var oe *oauth.Error
	if errors.As(err, &oe) && oe.Code != "server_error" {
		api.WriteMessage(w, http.StatusBadRequest, oe.Description)
		return
	}
	api.WriteMessage(w, http.StatusInternalServerError, err.Error())
}
