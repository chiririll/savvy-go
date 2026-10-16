package oauthapi

import (
	"encoding/json"
	"net"
	"net/http"

	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/oauth"
)

// oauthRegister is dynamic client registration (RFC 7591) for public clients.
func (h Handler) oauthRegister(w http.ResponseWriter, r *http.Request) {
	if _, open := h.OAuth.RegistrationOpenUntil(); !open {
		api.WriteMessage(w, http.StatusNotFound, "Not found.")
		return
	}
	var body struct {
		ClientName   string   `json:"client_name"`
		RedirectURIs []string `json:"redirect_uris"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, oauthBodyLimit)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeOAuthError(w, &oauth.Error{Code: "invalid_client_metadata", Description: "The body must be a JSON object."})
		return
	}
	c, err := h.OAuth.Register(r.Context(), body.ClientName, body.RedirectURIs, registrationInfo(r))
	if err != nil {
		writeOAuthError(w, err)
		return
	}
	api.OpenCORS(w)
	w.Header().Set("Cache-Control", "no-store")
	api.WriteJSON(w, http.StatusCreated, map[string]any{
		"client_id":                  c.ID,
		"client_name":                c.Name,
		"redirect_uris":              c.RedirectURIs,
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
	})
}

// registrationInfo is what the admin sees when deciding on a new app. RealIP
// middleware has already put the forwarded address in RemoteAddr.
func registrationInfo(r *http.Request) oauth.RequestInfo {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	return oauth.RequestInfo{IP: ip, UserAgent: r.UserAgent(), Origin: r.Header.Get("Origin")}
}
