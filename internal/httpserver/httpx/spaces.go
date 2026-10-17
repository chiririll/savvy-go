package httpx

import (
	"errors"
	"net/http"
	"strings"

	"savvy-go/internal/auth"
	"savvy-go/internal/spaces"
)

// WriteSpaceError maps space rules to 422 and forbidden ones to 403.
func WriteSpaceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, spaces.ErrNotMember):
		WriteMessage(w, http.StatusNotFound, "Not a member of this space.")
	case errors.Is(err, spaces.ErrInvitationInvalid):
		WriteMessage(w, http.StatusNotFound, err.Error())
	case errors.Is(err, spaces.ErrRegistrationClosed):
		WriteMessage(w, http.StatusForbidden, err.Error())
	case errors.Is(err, spaces.ErrInvitationCooldown):
		WriteMessage(w, http.StatusTooManyRequests, err.Error())
	case errors.Is(err, spaces.ErrGuestCannotAdmin), errors.Is(err, spaces.ErrLastAdmin), errors.Is(err, spaces.ErrLastMember),
		errors.Is(err, spaces.ErrSpaceLimit), errors.Is(err, spaces.ErrInvitationEmail),
		errors.Is(err, spaces.ErrInvitationExists), errors.Is(err, spaces.ErrAlreadyMember):
		WriteMessage(w, http.StatusUnprocessableEntity, err.Error())
	default:
		WriteMessage(w, http.StatusInternalServerError, err.Error())
	}
}

// SpaceName trims a space name and checks its length; it answers 422 when it is bad.
func SpaceName(w http.ResponseWriter, raw string) (string, bool) {
	name := strings.TrimSpace(raw)
	if name == "" || len(name) > 100 {
		WriteValidation(w, map[string][]string{"name": {"The name is required and must be at most 100 characters."}})
		return "", false
	}
	return name, true
}

// WriteChallengeError answers a failed second-factor challenge issue: 429 while
// the account waits after too many wrong codes, 500 otherwise.
func WriteChallengeError(w http.ResponseWriter, err error) {
	if errors.Is(err, auth.ErrLocked) {
		WriteMessage(w, http.StatusTooManyRequests, "Too many wrong verification codes. Try again later.")
		return
	}
	WriteMessage(w, http.StatusInternalServerError, err.Error())
}
