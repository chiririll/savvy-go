package oauthapi

import (
	"net/http"

	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/oauth"
)

// oauthToken is the token endpoint for the authorization_code and
// refresh_token grants. Clients are public: they prove themselves with PKCE.
func (h Handler) oauthToken(w http.ResponseWriter, r *http.Request) {
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
	c, err := h.OAuth.Client(r.Context(), f.Get("client_id"))
	if err == nil && c == nil {
		err = &oauth.Error{Code: "invalid_client", Description: "Unknown client."}
	}
	var ts *oauth.TokenSet
	if err == nil {
		if grant == "authorization_code" {
			ts, err = h.OAuth.Exchange(r.Context(), c.ID, f.Get("code"), f.Get("code_verifier"), f.Get("redirect_uri"))
		} else {
			ts, err = h.OAuth.Refresh(r.Context(), c.ID, f.Get("refresh_token"))
		}
	}
	if err != nil {
		writeOAuthError(w, err)
		return
	}
	api.OpenCORS(w)
	w.Header().Set("Cache-Control", "no-store")
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"access_token":  ts.AccessToken,
		"token_type":    "Bearer",
		"expires_in":    ts.ExpiresIn,
		"refresh_token": ts.RefreshToken,
	})
}
