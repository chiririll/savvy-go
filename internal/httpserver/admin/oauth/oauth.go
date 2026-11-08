// Package oauth serves the server admin's controls for OAuth clients: the registration
// window and the apps that registered.
package oauth

import (
	"errors"
	"net/http"
	"strings"
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
	// Kind is how the app came to be known: dcr, cimd or manual.
	Kind         string `json:"kind"`
	Confidential bool   `json:"confidential"`
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
			Kind: c.Kind, Confidential: c.SecretHash != "",
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

// Own clients: the admin creates the client and hands its id (and secret, shown
// once) to the app, e.g. in Claude's advanced connector settings.

type createClientRequest struct {
	Name         string   `json:"name"`
	RedirectURIs []string `json:"redirect_uris"`
	Confidential bool     `json:"confidential"`
}

// createdClientBody carries the secret, the only time it is ever shown.
type createdClientBody struct {
	clientBody
	ClientSecret string `json:"client_secret,omitempty"`
}

func (h Handler) oauthClientStore(w http.ResponseWriter, r *http.Request) {
	var body createClientRequest
	if !httpx.DecodeBody(w, r, &body) {
		return
	}
	c, secret, err := h.OAuth.CreateClient(r.Context(), body.Name, body.RedirectURIs, body.Confidential, httpx.User(r))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.WriteData(w, http.StatusCreated, createdClientBody{
		clientBody: clientBody{
			ClientID: c.ID, Name: c.Name, RedirectURIs: c.RedirectURIs, RegisteredAt: c.CreatedAt,
			Kind: c.Kind, Confidential: c.SecretHash != "",
		},
		ClientSecret: secret,
	})
}

// Metadata document hosts: apps that identify themselves with a URL are
// accepted only from hosts on this list, which starts empty.

type hostBody struct {
	Host string `json:"host"`
}

func (h Handler) oauthHostsIndex(w http.ResponseWriter, r *http.Request) {
	hosts, err := h.OAuth.CIMDHosts(r.Context())
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(hosts, func(h string) hostBody { return hostBody{Host: h} }))
}

func (h Handler) oauthHostStore(w http.ResponseWriter, r *http.Request) {
	var body hostBody
	if !httpx.DecodeBody(w, r, &body) {
		return
	}
	if err := h.OAuth.AddCIMDHost(r.Context(), body.Host); err != nil {
		writeError(w, err)
		return
	}
	httpx.WriteData(w, http.StatusCreated, hostBody{Host: strings.ToLower(strings.TrimSpace(body.Host))})
}

func (h Handler) oauthHostDestroy(w http.ResponseWriter, r *http.Request) {
	ok, err := h.OAuth.RemoveCIMDHost(r.Context(), chi.URLParam(r, "host"))
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		httpx.WriteMessage(w, http.StatusNotFound, "No such host.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeError answers a rule violation as 422 and anything else as 500.
func writeError(w http.ResponseWriter, err error) {
	var oe *oauth.Error
	if errors.As(err, &oe) && oe.Code != "server_error" {
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, oe.Description)
		return
	}
	httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
}
