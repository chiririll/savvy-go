package oauthapi

import (
	"net/http"
	"time"

	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/oauth"

	"github.com/go-chi/chi/v5"
)

// Registration window: an admin opens it from Administration → Security.

func registrationState(until time.Time, open bool) map[string]any {
	if !open {
		return map[string]any{"open": false, "expires_at": nil}
	}
	return map[string]any{"open": true, "expires_at": until.UTC()}
}

func (h Handler) oauthRegistrationShow(w http.ResponseWriter, r *http.Request) {
	until, open := h.OAuth.RegistrationOpenUntil()
	api.WriteData(w, http.StatusOK, registrationState(until, open))
}

func (h Handler) oauthRegistrationOpen(w http.ResponseWriter, r *http.Request) {
	until := h.OAuth.OpenRegistration(oauth.RegistrationWindow)
	api.WriteData(w, http.StatusOK, registrationState(until, true))
}

func (h Handler) oauthRegistrationClose(w http.ResponseWriter, r *http.Request) {
	h.OAuth.CloseRegistration()
	api.WriteData(w, http.StatusOK, registrationState(time.Time{}, false))
}

// Registered apps: the admin sees where each registration came from and can
// disconnect an app. Access itself is the user's own consent.

func (h Handler) oauthClientsIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.OAuth.Clients(r.Context())
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]map[string]any, len(list))
	for i, c := range list {
		out[i] = map[string]any{
			"client_id": c.ID, "name": c.Name, "redirect_uris": c.RedirectURIs,
			"ip": c.Request.IP, "user_agent": c.Request.UserAgent, "origin": c.Request.Origin,
			"registered_at": c.CreatedAt,
		}
	}
	api.WriteData(w, http.StatusOK, out)
}

func (h Handler) oauthClientDestroy(w http.ResponseWriter, r *http.Request) {
	ok, err := h.OAuth.Remove(r.Context(), chi.URLParam(r, "id"), api.User(r))
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		api.WriteMessage(w, http.StatusNotFound, "No such app.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
