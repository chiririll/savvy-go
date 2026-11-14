// Package emailverify serves the routes that confirm email addresses.
package emailverify

import (
	"encoding/json"
	"errors"
	"net/http"

	"savvy-go/internal/auth"
	"savvy-go/internal/httpserver/httpx"
)

// verifyEmailResend sends the signed-in user a new confirmation link.
func (h Handler) verifyEmailResend(w http.ResponseWriter, r *http.Request) {
	u := httpx.User(r)
	if u.EmailVerified {
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, "Your email address is already confirmed.")
		return
	}
	if problem := h.Verify.Send(r.Context(), httpx.Locale(r), u); problem != "" {
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, problem)
		return
	}
	httpx.WriteMessage(w, http.StatusOK, "Confirmation email sent.")
}

// verifyEmail confirms the address named by the emailed token. It needs no
// session: the link may be opened on another device.
func (h Handler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	var viewerID int64
	if u := h.Guard.SessionUser(r); u != nil {
		viewerID = u.ID
	}
	ok, err := h.Verify.Confirm(r.Context(), body.Token, viewerID)
	if errors.Is(err, auth.ErrOtherAccount) {
		httpx.WriteMessage(w, http.StatusForbidden, "This link belongs to another account.")
		return
	}
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		httpx.WriteValidation(w, map[string][]string{"token": {"This link is invalid or has expired."}})
		return
	}
	httpx.WriteMessage(w, http.StatusOK, "Email address confirmed.")
}
