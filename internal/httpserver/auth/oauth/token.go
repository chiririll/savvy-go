package oauth

import (
	"errors"
	"net/http"
	"net/url"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/oauth"
)

// clientCredentials reads the client id and secret from HTTP Basic (RFC 6749
// §2.3.1, form-encoded values) or from the body. A Basic header wins; a body
// client_id that disagrees with it is an error.
func clientCredentials(r *http.Request) (id, secret string, err error) {
	id, secret = r.PostForm.Get("client_id"), r.PostForm.Get("client_secret")
	user, pass, ok := r.BasicAuth()
	if !ok {
		return id, secret, nil
	}
	if user, err = url.QueryUnescape(user); err != nil {
		return "", "", err
	}
	if pass, err = url.QueryUnescape(pass); err != nil {
		return "", "", err
	}
	if id != "" && id != user {
		return "", "", errors.New("client_id differs from the Authorization header")
	}
	return user, pass, nil
}

// oauthToken is the token endpoint for the authorization_code and
// refresh_token grants. Public clients prove themselves with PKCE; clients an
// admin created with a secret must send it as well, for both grants.
func (h Handler) oauthToken(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, oauthBodyLimit)
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, &oauth.Error{Code: "invalid_request", Description: "The body must be form-encoded."})
		return
	}
	f := r.PostForm
	grant := f.Get("grant_type")
	if grant != "authorization_code" && grant != "refresh_token" {
		writeOAuthError(
			w,
			&oauth.Error{Code: "unsupported_grant_type", Description: "Use authorization_code or refresh_token."},
		)
		return
	}
	var c *oauth.Client
	id, secret, err := clientCredentials(r)
	if err != nil {
		err = &oauth.Error{Code: "invalid_client", Description: "Bad client credentials."}
	} else {
		c, err = h.OAuth.AuthenticateClient(r.Context(), id, secret)
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
