package httpserver

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (s *Server) webauthnIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.webauthn.List(r.Context(), userFrom(r).ID)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	data := make([]any, 0, len(list))
	for _, c := range list {
		data = append(data, c.JSON())
	}
	writeJSON(w, http.StatusOK, map[string]any{"credentials": data})
}

func (s *Server) webauthnRegisterOptions(w http.ResponseWriter, r *http.Request) {
	token, options, err := s.webauthn.BeginRegistration(r.Context(), userFrom(r))
	if err != nil {
		if err.Error() == "limit" {
			writeMessage(w, http.StatusUnprocessableEntity, "You can register at most 20 passkeys.")
			return
		}
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "options": options})
}

func (s *Server) webauthnRegisterVerify(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token    string          `json:"token"`
		Name     string          `json:"name"`
		Response json.RawMessage `json:"response"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" || len(body.Response) == 0 {
		writeValidation(w, map[string][]string{"response": {"The given data was invalid."}})
		return
	}
	cred, err := s.webauthn.FinishRegistration(r.Context(), userFrom(r), body.Token, body.Name, body.Response)
	if err != nil {
		switch err.Error() {
		case "challenge":
			writeMessage(w, http.StatusUnprocessableEntity, "Passkey challenge is invalid or has expired.")
		default:
			writeMessage(w, http.StatusUnprocessableEntity, "Passkey verification failed.")
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"message":    "Passkey registered.",
		"credential": cred.JSON(),
	})
}

func (s *Server) webauthnUpdate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeValidation(w, map[string][]string{"name": {"The given data was invalid."}})
		return
	}
	if err := s.webauthn.Rename(r.Context(), userFrom(r).ID, id, body.Name); err != nil {
		if err == sql.ErrNoRows {
			writeMessage(w, http.StatusNotFound, "Not found.")
			return
		}
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Passkey renamed."})
}

func (s *Server) webauthnDestroy(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := s.webauthn.Delete(r.Context(), userFrom(r).ID, id); err != nil {
		if err == sql.ErrNoRows {
			writeMessage(w, http.StatusNotFound, "Not found.")
			return
		}
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Passkey removed."})
}

func (s *Server) webauthnLoginOptions(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TwoFactorToken *string `json:"two_factor_token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	var userID *int64
	if body.TwoFactorToken != nil && *body.TwoFactorToken != "" {
		u, err := s.challenges.Resolve(r.Context(), *body.TwoFactorToken)
		if err != nil || u == nil {
			writeMessage(w, http.StatusUnauthorized, "Invalid or expired token.")
			return
		}
		userID = &u.ID
		u2 := u
		token, options, err := s.webauthn.BeginLogin(r.Context(), u2)
		if err != nil {
			writeMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"token": token, "options": options})
		return
	}
	_ = userID
	token, options, err := s.webauthn.BeginLogin(r.Context(), nil)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "options": options})
}

func (s *Server) webauthnLoginVerify(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token          string          `json:"token"`
		Response       json.RawMessage `json:"response"`
		TwoFactorToken *string         `json:"two_factor_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" || len(body.Response) == 0 {
		writeValidation(w, map[string][]string{"response": {"The given data was invalid."}})
		return
	}
	u, err := s.webauthn.FinishLogin(r.Context(), body.Token, body.Response)
	if err != nil {
		switch err.Error() {
		case "challenge":
			writeMessage(w, http.StatusUnprocessableEntity, "Passkey challenge is invalid or has expired.")
		default:
			writeMessage(w, http.StatusUnprocessableEntity, "Passkey verification failed.")
		}
		return
	}
	if body.TwoFactorToken != nil && *body.TwoFactorToken != "" {
		pending, err := s.challenges.Resolve(r.Context(), *body.TwoFactorToken)
		if err != nil || pending == nil || pending.ID != u.ID {
			writeMessage(w, http.StatusUnauthorized, "Invalid or expired token.")
			return
		}
		_, _ = s.challenges.Consume(r.Context(), *body.TwoFactorToken)
	}
	s.issueSession(w, r, u, http.StatusOK, true)
}
