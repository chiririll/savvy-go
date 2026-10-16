// Package api is what the domain packages of the HTTP server share: JSON
// replies and the space a request works in.
package api

import (
	"encoding/json"
	"net/http"
	"time"
)

type Envelope struct {
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
	Meta    any    `json:"meta,omitempty"`
}

type validationBody struct {
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func WriteData(w http.ResponseWriter, status int, data any) {
	WriteJSON(w, status, Envelope{Data: data})
}

func WriteDataMsg(w http.ResponseWriter, status int, data any, message string) {
	WriteJSON(w, status, Envelope{Data: data, Message: message})
}

func WriteMessage(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"message": message})
}

func WriteValidation(w http.ResponseWriter, errors map[string][]string) {
	msg := "The given data was invalid."
	if len(errors) == 1 {
		for _, msgs := range errors {
			if len(msgs) > 0 {
				msg = msgs[0]
				break
			}
		}
	}
	WriteJSON(w, http.StatusUnprocessableEntity, validationBody{Message: msg, Errors: errors})
}

func WriteHealth(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/health+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// OpenCORS lets any origin call the endpoint without cookies, as OAuth and MCP
// clients running in a browser do.
func OpenCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "WWW-Authenticate, Mcp-Session-Id")
}

// TimeOrNil renders a timestamp as RFC 3339 UTC, or null.
func TimeOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}
