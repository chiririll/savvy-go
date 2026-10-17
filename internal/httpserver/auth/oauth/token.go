package oauth

import (
	"net/http"

	"savvy-go/internal/httpserver/httpx"
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
	httpx.OpenCORS(w)
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, tokenResponse{
		AccessToken: ts.AccessToken, TokenType: "Bearer", ExpiresIn: ts.ExpiresIn, RefreshToken: ts.RefreshToken,
	})
}

// tokenResponse is the successful token response of RFC 6749.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}
