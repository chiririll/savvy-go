// Package oauth serves the server admin's controls for OAuth clients: the registration
// window and the apps that registered.
package oauth

import (
	"net/http"
	"time"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/oauth"

	"github.com/go-chi/chi/v5"
)

// Registration window: an admin opens it from Administration → Security.

// registrationBody says whether dynamic client registration is open, and until when.
type registrationBody struct {
	Open      bool       `json:"open"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func registrationState(until time.Time, open bool) registrationBody {
	if !open {
		return registrationBody{}
	}
	u := until.UTC()
	return registrationBody{Open: true, ExpiresAt: &u}
}

// clientBody is a registered app as the admin sees it. Its keys follow RFC 7591.
type clientBody struct {
	ClientID     string    `json:"client_id"`
	Name         string    `json:"name"`
	RedirectURIs []string  `json:"redirect_uris"`
	IP           string    `json:"ip"`
	UserAgent    string    `json:"user_agent"`
	Origin       string    `json:"origin"`
	RegisteredAt time.Time `json:"registered_at"`
}

func (h Handler) oauthRegistrationShow(w http.ResponseWriter, r *http.Request) {
	until, open := h.OAuth.RegistrationOpenUntil()
	httpx.WriteData(w, http.StatusOK, registrationState(until, open))
}

func (h Handler) oauthRegistrationOpen(w http.ResponseWriter, r *http.Request) {
	until := h.OAuth.OpenRegistration(oauth.RegistrationWindow)
	httpx.WriteData(w, http.StatusOK, registrationState(until, true))
}

func (h Handler) oauthRegistrationClose(w http.ResponseWriter, r *http.Request) {
	h.OAuth.CloseRegistration()
	httpx.WriteData(w, http.StatusOK, registrationState(time.Time{}, false))
}

// Registered apps: the admin sees where each registration came from and can
// disconnect an app. Access itself is the user's own consent.

func (h Handler) oauthClientsIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.OAuth.Clients(r.Context())
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(list, func(c oauth.Client) clientBody {
		return clientBody{
			ClientID: c.ID, Name: c.Name, RedirectURIs: c.RedirectURIs,
			IP: c.Request.IP, UserAgent: c.Request.UserAgent, Origin: c.Request.Origin, RegisteredAt: c.CreatedAt,
		}
	}))
}

func (h Handler) oauthClientDestroy(w http.ResponseWriter, r *http.Request) {
	ok, err := h.OAuth.Remove(r.Context(), chi.URLParam(r, "id"), httpx.User(r))
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		httpx.WriteMessage(w, http.StatusNotFound, "No such app.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
