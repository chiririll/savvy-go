package oauth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/confirm"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/oauth"
)

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
func (h Handler) oauthAuthorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	redirectURI := q.Get("redirect_uri")
	if _, err := h.OAuth.CheckAuthorize(r.Context(), q.Get("client_id"), redirectURI); err != nil {
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
		http.Redirect(
			w,
			r,
			redirectWith(redirectURI, map[string]string{"error": code, "error_description": desc, "state": state}),
			http.StatusFound,
		)
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

// oauthConsentInfo tells the consent screen which app is asking, what it will
// be able to read, and which of the user's spaces it can be given.
func (h Handler) oauthConsentInfo(w http.ResponseWriter, r *http.Request) {
	c, err := h.OAuth.CheckAuthorize(r.Context(), r.URL.Query().Get("client_id"), r.URL.Query().Get("redirect_uri"))
	if err != nil {
		writeOAuthAPIError(w, err)
		return
	}
	mine, err := h.Spaces.ForUser(r.Context(), httpx.User(r).ID)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, consentBody{ClientName: c.Name, Spaces: dto.Map(mine, dto.NewSpace)})
}

// oauthConsent records the user's decision and says where to send the browser.
func (h Handler) oauthConsent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClientID            string   `json:"client_id"`
		RedirectURI         string   `json:"redirect_uri"`
		State               string   `json:"state"`
		CodeChallenge       string   `json:"code_challenge"`
		CodeChallengeMethod string   `json:"code_challenge_method"`
		Approve             bool     `json:"approve"`
		AllSpaces           bool     `json:"all_spaces"`
		SpaceIDs            []int64  `json:"space_ids"`
		Permissions         []string `json:"permissions"`
		confirm.Fields
	}
	r.Body = http.MaxBytesReader(w, r.Body, oauthBodyLimit)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteMessage(w, http.StatusBadRequest, "The given data was invalid.")
		return
	}
	if !body.Approve {
		if _, err := h.OAuth.CheckAuthorize(r.Context(), body.ClientID, body.RedirectURI); err != nil {
			writeOAuthAPIError(w, err)
			return
		}
		httpx.WriteData(
			w,
			http.StatusOK,
			map[string]string{
				"redirect_url": redirectWith(body.RedirectURI, map[string]string{"error": "access_denied", "state": body.State}),
			},
		)
		return
	}
	// The app gets only the spaces the user picked, and only ones they belong to.
	if body.AllSpaces == (len(body.SpaceIDs) > 0) {
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, "Choose all spaces, or at least one space.")
		return
	}
	for _, id := range body.SpaceIDs {
		if role, err := h.Spaces.Role(r.Context(), id, httpx.User(r).ID); err != nil || role == "" {
			httpx.WriteMessage(w, http.StatusUnprocessableEntity, "You can only choose spaces you belong to.")
			return
		}
	}
	// Approving mints the app a token, so it is asked last, like issuing one: a bad form costs no attempt.
	if !h.Confirm.Require(w, r, body.Fields) {
		return
	}
	code, err := h.OAuth.IssueCode(r.Context(), oauth.CodeRequest{
		ClientID: body.ClientID, RedirectURI: body.RedirectURI, CodeChallenge: body.CodeChallenge,
		CodeChallengeMethod: body.CodeChallengeMethod, User: httpx.User(r),
		AllSpaces: body.AllSpaces, SpaceIDs: body.SpaceIDs, Permissions: body.Permissions,
	})
	if err != nil {
		writeOAuthAPIError(w, err)
		return
	}
	httpx.WriteData(
		w,
		http.StatusOK,
		map[string]string{
			"redirect_url": redirectWith(body.RedirectURI, map[string]string{"code": code, "state": body.State}),
		},
	)
}

// consentBody is what the consent screen shows: the app and the spaces it may be given.
type consentBody struct {
	ClientName string      `json:"client_name"`
	Spaces     []dto.Space `json:"spaces"`
}
