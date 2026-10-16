// Package webauthn serves passkey registration and sign-in.
package webauthn

import (
	"encoding/json"
	"net/http"
	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/guard"
	"savvy-go/internal/httpserver/signin"
	"strconv"

	"savvy-go/internal/auth"
	"savvy-go/internal/httpserver/dto"

	"github.com/go-chi/chi/v5"
)

// Handler holds what the passkey routes need.
type Handler struct {
	WebAuthn   auth.WebAuthn
	Challenges auth.Challenges
	Login      signin.Service
	Guard      *guard.Guard
}

func (h Handler) webauthnIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.WebAuthn.List(r.Context(), api.User(r).ID)
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"credentials": dto.Map(list, dto.NewWebAuthnCred)})
}

func (h Handler) webauthnRegisterOptions(w http.ResponseWriter, r *http.Request) {
	token, options, err := h.WebAuthn.BeginRegistration(r.Context(), api.User(r))
	if err != nil {
		if err.Error() == "limit" {
			api.WriteMessage(w, http.StatusUnprocessableEntity, "You can register at most 20 passkeys.")
			return
		}
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"token": token, "options": options})
}

func (h Handler) webauthnRegisterVerify(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token    string          `json:"token"`
		Name     string          `json:"name"`
		Response json.RawMessage `json:"response"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" || len(body.Response) == 0 {
		api.WriteValidation(w, map[string][]string{"response": {"The given data was invalid."}})
		return
	}
	cred, err := h.WebAuthn.FinishRegistration(r.Context(), api.User(r), body.Token, body.Name, body.Response)
	if err != nil {
		switch err.Error() {
		case "challenge":
			api.WriteMessage(w, http.StatusUnprocessableEntity, "Passkey challenge is invalid or has expired.")
		default:
			api.WriteMessage(w, http.StatusUnprocessableEntity, "Passkey verification failed.")
		}
		return
	}
	api.WriteJSON(w, http.StatusCreated, map[string]any{
		"message":    "Passkey registered.",
		"credential": dto.NewWebAuthnCred(*cred),
	})
}

func (h Handler) webauthnUpdate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		api.WriteValidation(w, map[string][]string{"name": {"The given data was invalid."}})
		return
	}
	if err := h.WebAuthn.Rename(r.Context(), api.User(r).ID, id, body.Name); err != nil {
		if err == auth.ErrNotFound {
			api.WriteMessage(w, http.StatusNotFound, "Not found.")
			return
		}
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"message": "Passkey renamed."})
}

func (h Handler) webauthnDestroy(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.WebAuthn.Delete(r.Context(), api.User(r).ID, id); err != nil {
		if err == auth.ErrNotFound {
			api.WriteMessage(w, http.StatusNotFound, "Not found.")
			return
		}
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"message": "Passkey removed."})
}

func (h Handler) webauthnLoginOptions(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TwoFactorToken *string `json:"two_factor_token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	var userID *int64
	if body.TwoFactorToken != nil && *body.TwoFactorToken != "" {
		u, err := h.Challenges.Resolve(r.Context(), *body.TwoFactorToken)
		if err != nil || u == nil {
			api.WriteMessage(w, http.StatusUnauthorized, "Invalid or expired token.")
			return
		}
		userID = &u.ID
		u2 := u
		token, options, err := h.WebAuthn.BeginLogin(r.Context(), u2)
		if err != nil {
			api.WriteMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
		api.WriteJSON(w, http.StatusOK, map[string]any{"token": token, "options": options})
		return
	}
	_ = userID
	token, options, err := h.WebAuthn.BeginLogin(r.Context(), nil)
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"token": token, "options": options})
}

func (h Handler) webauthnLoginVerify(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token          string          `json:"token"`
		Response       json.RawMessage `json:"response"`
		TwoFactorToken *string         `json:"two_factor_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" || len(body.Response) == 0 {
		api.WriteValidation(w, map[string][]string{"response": {"The given data was invalid."}})
		return
	}
	u, err := h.WebAuthn.FinishLogin(r.Context(), body.Token, body.Response)
	if err != nil {
		switch err.Error() {
		case "challenge":
			api.WriteMessage(w, http.StatusUnprocessableEntity, "Passkey challenge is invalid or has expired.")
		default:
			api.WriteMessage(w, http.StatusUnprocessableEntity, "Passkey verification failed.")
		}
		return
	}
	if body.TwoFactorToken != nil && *body.TwoFactorToken != "" {
		pending, err := h.Challenges.Resolve(r.Context(), *body.TwoFactorToken)
		if err != nil || pending == nil || pending.ID != u.ID {
			api.WriteMessage(w, http.StatusUnauthorized, "Invalid or expired token.")
			return
		}
		_, _ = h.Challenges.Consume(r.Context(), *body.TwoFactorToken)
	}
	h.Login.Issue(w, r, u, http.StatusOK, true)
}
