// Package confirm asks the signed-in user to prove who they are again before a
// change that would give someone lasting access to the account: a new passkey,
// 2FA, an API token. A stolen session alone then cannot plant one.
package confirm

import (
	"encoding/json"
	"net/http"

	"savvy-go/internal/auth"
	"savvy-go/internal/auth/password"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/guard"
)

// Fields are what a request carries to confirm itself. They sit in the body of the
// change, so there is no state between asking and acting.
type Fields struct {
	CurrentPassword string `json:"current_password"`
	Code            string `json:"code"`
}

// Confirmer checks Fields against the signed-in user.
type Confirmer struct {
	TwoFactor auth.TwoFactor
	Guard     *guard.Guard
}

// RequireBody is Require for a route that takes nothing else: the fields are the
// whole body, and an empty body is the same as giving nothing.
func (c Confirmer) RequireBody(w http.ResponseWriter, r *http.Request) bool {
	var f Fields
	_ = json.NewDecoder(r.Body).Decode(&f)
	return c.Require(w, r, f)
}

// Require reports whether the request may go on. When it may not it has already
// answered: 403 confirmation_required when nothing was given, 422
// confirmation_invalid when it was wrong, 429 when the account has had too many
// attempts. Every attempt counts against the account's sign-in limit, so guessing
// here gains nothing over guessing at the login form.
//
// A user with 2FA confirms with an authenticator or recovery code, which the
// password alone does not replace. Otherwise the password does. An account with
// neither (single sign-on only) has nothing to confirm with and passes.
func (c Confirmer) Require(w http.ResponseWriter, r *http.Request, f Fields) bool {
	u := httpx.User(r)
	if u == nil {
		httpx.WriteMessage(w, http.StatusUnauthorized, "Unauthenticated.")
		return false
	}
	byCode := u.HasTwoFactor()
	byPassword := !byCode && !u.IsSSOOnly && u.Password != nil
	if !byCode && !byPassword {
		return true
	}
	given := f.CurrentPassword
	if byCode {
		given = f.Code
	}
	if given == "" {
		message := "Enter your password to continue."
		if byCode {
			message = "Enter your authentication code to continue."
		}
		httpx.WriteCoded(w, http.StatusForbidden, message, "confirmation_required")
		return false
	}
	if !c.Guard.AllowAccount(w, u.Email) {
		return false
	}
	var ok bool
	if byCode {
		ok = c.TwoFactor.VerifyAny(r.Context(), u, given)
	} else {
		ok = password.Check(*u.Password, given)
	}
	if !ok {
		httpx.WriteCoded(w, http.StatusUnprocessableEntity, "That is not right. Try again.", "confirmation_invalid")
		return false
	}
	return true
}
