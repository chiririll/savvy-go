package httpserver

import (
	"encoding/json"
	"net/http"
)

type envelope struct {
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
	Meta    any    `json:"meta,omitempty"`
}

type validationBody struct {
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func writeData(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, envelope{Data: data})
}

func writeDataMsg(w http.ResponseWriter, status int, data any, message string) {
	writeJSON(w, status, envelope{Data: data, Message: message})
}

func writeMessage(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"message": message})
}

func writeValidation(w http.ResponseWriter, errors map[string][]string) {
	msg := "The given data was invalid."
	if len(errors) == 1 {
		for _, msgs := range errors {
			if len(msgs) > 0 {
				msg = msgs[0]
				break
			}
		}
	}
	writeJSON(w, http.StatusUnprocessableEntity, validationBody{Message: msg, Errors: errors})
}

func writeHealth(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/health+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
