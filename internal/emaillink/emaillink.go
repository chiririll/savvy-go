// Package emaillink signs the links Go Savvy sends by email, so a link proves
// it came from this server without a table of issued links. It is a leaf: it
// knows nothing of users, spaces or HTTP.
package emaillink

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"savvy-go/internal/signing"
)

// Each purpose signs a different prefix, so a signature made for one cannot be
// replayed for the other.
const (
	purposeVerify = "email-verification"
	purposeInvite = "invitation-email"
)

var enc = base64.RawURLEncoding

// Signer signs with the current server key. A restore that replaces the key
// ends every link sent before it.
type Signer struct {
	Keys *signing.Holder
}

func (s Signer) sign(parts ...string) string {
	return enc.EncodeToString(s.Keys.Key().Sign([]byte(strings.Join(parts, "\x00"))))
}

func (s Signer) check(sig string, parts ...string) bool {
	raw, err := enc.DecodeString(sig)
	if err != nil {
		return false
	}
	return signing.Verify(s.Keys.Key().Public(), []byte(strings.Join(parts, "\x00")), raw)
}

func norm(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

// Verification is the token of a "confirm your email" link: the user, the
// expiry and a signature over both and the address. It is not single-use;
// confirming twice changes nothing.
func (s Signer) Verification(userID int64, email string, expires time.Time) string {
	id, exp := strconv.FormatInt(userID, 10), strconv.FormatInt(expires.Unix(), 10)
	return id + "." + exp + "." + s.sign(purposeVerify, id, exp, norm(email))
}

func splitVerification(token string) (id, exp, sig string, ok bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

// VerificationUser is the user a token names. The token is still unproven:
// follow with VerificationValid and that user's current address.
func (s Signer) VerificationUser(token string) (int64, bool) {
	id, _, _, ok := splitVerification(token)
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseInt(id, 10, 64)
	return n, err == nil && n > 0
}

// VerificationValid reports whether token was signed by this server for the
// user's current address and has not expired at now.
func (s Signer) VerificationValid(token, email string, now time.Time) bool {
	id, exp, sig, ok := splitVerification(token)
	if !ok {
		return false
	}
	unix, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || !now.Before(time.Unix(unix, 0)) {
		return false
	}
	return s.check(sig, purposeVerify, id, exp, norm(email))
}

// Invite signs an invitation token for an address. Only the emailed link
// carries it, so a link copied from the app has none and proves nothing.
func (s Signer) Invite(inviteToken, email string) string {
	return s.sign(purposeInvite, inviteToken, norm(email))
}

// InviteValid reports whether sig was made by Invite for this token and address.
func (s Signer) InviteValid(inviteToken, email, sig string) bool {
	return sig != "" && s.check(sig, purposeInvite, inviteToken, norm(email))
}
