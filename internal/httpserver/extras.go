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
