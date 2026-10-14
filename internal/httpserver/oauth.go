package httpserver

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/oauth"
)

// The OAuth 2.1 authorization server MCP clients (Claude web) sign in with.
// Endpoints sit outside /api, at the paths the clients discover through
// /.well-known; the consent screen is an SPA page that talks to
// /api/oauth/authorize with the user's session. See internal/oauth.

const oauthBodyLimit = 64 << 10

// oauthScopes: access tokens are read-only API tokens.
var oauthScopes = []string{"read"}

// mcpRoutes mounts the OAuth endpoints and the MCP endpoint they protect.
func (s *Server) mcpRoutes(r chi.Router) {
	for _, p := range []string{
		"/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/*",
		"/.well-known/oauth-authorization-server", "/oauth/register", "/oauth/token", "/mcp",
	} {
		r.Options(p, oauthPreflight)
	}
	r.Get("/.well-known/oauth-protected-resource", s.oauthResourceMetadata)
	r.Get("/.well-known/oauth-protected-resource/*", s.oauthResourceMetadata)
	r.Get("/.well-known/oauth-authorization-server", s.oauthServerMetadata)
	r.Post("/oauth/register", s.oauthRegister)
	r.Get("/oauth/authorize", s.oauthAuthorize)
	r.Post("/oauth/token", s.oauthToken)
	r.Handle("/mcp", s.mcpHandler())
}

// oauthPreflight answers CORS preflights from browser-based MCP clients. The
// endpoints take no cookies, so any origin may call them.
func oauthPreflight(w http.ResponseWriter, r *http.Request) {
	oauthCORS(w)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Mcp-Session-Id, Mcp-Protocol-Version, Last-Event-ID")
	w.WriteHeader(http.StatusNoContent)
}

func oauthCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "WWW-Authenticate, Mcp-Session-Id")
}

// mcpResourceURL is the identifier of the MCP endpoint, what clients bind tokens to.
func (s *Server) mcpResourceURL(r *http.Request) string { return s.appURL(r.Context()) + "/mcp" }

func (s *Server) oauthResourceMetadata(w http.ResponseWriter, r *http.Request) {
	oauthCORS(w)
	base := s.appURL(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"resource":                 s.mcpResourceURL(r),
		"authorization_servers":    []string{base},
		"scopes_supported":         oauthScopes,
		"bearer_methods_supported": []string{"header"},
		"resource_name":            "Go Savvy",
	})
}

func (s *Server) oauthServerMetadata(w http.ResponseWriter, r *http.Request) {
	oauthCORS(w)
	base := s.appURL(r.Context())
	meta := map[string]any{
		"issuer":                                base,
		"authorization_endpoint":                base + "/oauth/authorize",
		"token_endpoint":                        base + "/oauth/token",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"scopes_supported":                      oauthScopes,
	}
	// Registration exists only while an admin has it open; closed, nothing
	// hints that it could be, and clients must not cache the answer.
	if _, open := s.oauth.RegistrationOpenUntil(); open {
		meta["registration_endpoint"] = base + "/oauth/register"
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, meta)
}

// writeOAuthError sends an RFC 6749 §5.2 error body.
func writeOAuthError(w http.ResponseWriter, err error) {
	oauthCORS(w)
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
	writeJSON(w, status, map[string]string{"error": oe.Code, "error_description": oe.Description})
}

// oauthRegister is dynamic client registration (RFC 7591) for public clients.
func (s *Server) oauthRegister(w http.ResponseWriter, r *http.Request) {
	if _, open := s.oauth.RegistrationOpenUntil(); !open {
		writeMessage(w, http.StatusNotFound, "Not found.")
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
	c, err := s.oauth.Register(r.Context(), body.ClientName, body.RedirectURIs, registrationInfo(r))
	if err != nil {
		writeOAuthError(w, err)
		return
	}
	oauthCORS(w)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, map[string]any{
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

// redirectWith adds params to a redirect URI, keeping any query it has.
func redirectWith(base string, params map[string]string) string {
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	q := u.Query()
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// oauthAuthorize is where a client sends the user's browser. A bad client or
// redirect URI is shown to the user, never redirected to; everything else
// goes on to the SPA's consent screen, or back to the client as an error.
func (s *Server) oauthAuthorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	redirectURI := q.Get("redirect_uri")
	if _, err := s.oauth.CheckAuthorize(r.Context(), q.Get("client_id"), redirectURI); err != nil {
		var oe *oauth.Error
		msg := "Authorization failed."
		if errors.As(err, &oe) {
			msg = oe.Description
		}
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	state := q.Get("state")
	fail := func(code, desc string) {
		http.Redirect(w, r, redirectWith(redirectURI, map[string]string{"error": code, "error_description": desc, "state": state}), http.StatusFound)
	}
	if q.Get("response_type") != "code" {
		fail("unsupported_response_type", "Only response_type=code is supported.")
		return
	}
	if q.Get("code_challenge") == "" || q.Get("code_challenge_method") != "S256" {
		fail("invalid_request", "PKCE with code_challenge_method=S256 is required.")
		return
	}
	http.Redirect(w, r, "/oauth/consent?"+r.URL.RawQuery, http.StatusFound)
}

// oauthConsentInfo tells the consent screen which app is asking.
func (s *Server) oauthConsentInfo(w http.ResponseWriter, r *http.Request) {
	c, err := s.oauth.CheckAuthorize(r.Context(), r.URL.Query().Get("client_id"), r.URL.Query().Get("redirect_uri"))
	if err != nil {
		writeOAuthAPIError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"client_name": c.Name})
}

// oauthConsent records the user's decision and says where to send the browser.
func (s *Server) oauthConsent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClientID            string `json:"client_id"`
		RedirectURI         string `json:"redirect_uri"`
		State               string `json:"state"`
		CodeChallenge       string `json:"code_challenge"`
		CodeChallengeMethod string `json:"code_challenge_method"`
		Approve             bool   `json:"approve"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, oauthBodyLimit)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeMessage(w, http.StatusBadRequest, "The given data was invalid.")
		return
	}
	if !body.Approve {
		if _, err := s.oauth.CheckAuthorize(r.Context(), body.ClientID, body.RedirectURI); err != nil {
			writeOAuthAPIError(w, err)
			return
		}
		writeData(w, http.StatusOK, map[string]string{"redirect_url": redirectWith(body.RedirectURI, map[string]string{"error": "access_denied", "state": body.State})})
		return
	}
	code, err := s.oauth.IssueCode(r.Context(), oauth.CodeRequest{
		ClientID: body.ClientID, RedirectURI: body.RedirectURI, CodeChallenge: body.CodeChallenge,
		CodeChallengeMethod: body.CodeChallengeMethod, User: userFrom(r),
	})
	if err != nil {
		writeOAuthAPIError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]string{"redirect_url": redirectWith(body.RedirectURI, map[string]string{"code": code, "state": body.State})})
}

// writeOAuthAPIError is writeOAuthError for the session API: the SPA reads message.
func writeOAuthAPIError(w http.ResponseWriter, err error) {
	var oe *oauth.Error
	if errors.As(err, &oe) && oe.Code != "server_error" {
		writeMessage(w, http.StatusBadRequest, oe.Description)
		return
	}
	writeMessage(w, http.StatusInternalServerError, err.Error())
}

// oauthToken is the token endpoint for the authorization_code and
// refresh_token grants. Clients are public: they prove themselves with PKCE.
func (s *Server) oauthToken(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, oauthBodyLimit)
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, &oauth.Error{Code: "invalid_request", Description: "The body must be form-encoded."})
		return
	}
	f := r.PostForm
	grant := f.Get("grant_type")
	if grant != "authorization_code" && grant != "refresh_token" {
		writeOAuthError(w, &oauth.Error{Code: "unsupported_grant_type", Description: "Use authorization_code or refresh_token."})
		return
	}
	c, err := s.oauth.Client(r.Context(), f.Get("client_id"))
	if err == nil && c == nil {
		err = &oauth.Error{Code: "invalid_client", Description: "Unknown client."}
	}
	var ts *oauth.TokenSet
	if err == nil {
		if grant == "authorization_code" {
			ts, err = s.oauth.Exchange(r.Context(), c.ID, f.Get("code"), f.Get("code_verifier"), f.Get("redirect_uri"))
		} else {
			ts, err = s.oauth.Refresh(r.Context(), c.ID, f.Get("refresh_token"))
		}
	}
	if err != nil {
		writeOAuthError(w, err)
		return
	}
	oauthCORS(w)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  ts.AccessToken,
		"token_type":    "Bearer",
		"expires_in":    ts.ExpiresIn,
		"refresh_token": ts.RefreshToken,
		"scope":         strings.Join(oauthScopes, " "),
	})
}

// Registration window: an admin opens it from Administration → Security.

func registrationState(until time.Time, open bool) map[string]any {
	if !open {
		return map[string]any{"open": false, "expires_at": nil}
	}
	return map[string]any{"open": true, "expires_at": until.UTC()}
}

func (s *Server) oauthRegistrationShow(w http.ResponseWriter, r *http.Request) {
	until, open := s.oauth.RegistrationOpenUntil()
	writeData(w, http.StatusOK, registrationState(until, open))
}

func (s *Server) oauthRegistrationOpen(w http.ResponseWriter, r *http.Request) {
	until := s.oauth.OpenRegistration(oauth.RegistrationWindow)
	writeData(w, http.StatusOK, registrationState(until, true))
}

func (s *Server) oauthRegistrationClose(w http.ResponseWriter, r *http.Request) {
	s.oauth.CloseRegistration()
	writeData(w, http.StatusOK, registrationState(time.Time{}, false))
}

// Registered apps: the admin sees where each registration came from and can
// disconnect an app. Access itself is the user's own consent.

func (s *Server) oauthClientsIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.oauth.Clients(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
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
	writeData(w, http.StatusOK, out)
}

func (s *Server) oauthClientDestroy(w http.ResponseWriter, r *http.Request) {
	ok, err := s.oauth.Remove(r.Context(), chi.URLParam(r, "id"), userFrom(r))
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeMessage(w, http.StatusNotFound, "No such app.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
