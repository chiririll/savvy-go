package smtp

import (
	"fmt"
	"strings"
	"time"
)

// The emails Go Savvy sends. Each takes plain values and returns a Message; the
// caller builds the link, so a message never decides what a link proves.

const expiryLayout = "2006-01-02 15:04 UTC"

// PasswordReset is the email with a one-time link to choose a new password.
func PasswordReset(to, name, link string, expires time.Time) Message {
	return Message{
		To:      to,
		Subject: "Set your Go Savvy password",
		Text: fmt.Sprintf("Hello %s,\n\nAn administrator asked to reset your Go Savvy password. "+
			"Open this link to choose a new one:\n\n%s\n\n"+
			"The link works once and expires on %s. If you did not expect this email, you can ignore it.\n",
			oneLine(name), link, expires.UTC().Format(expiryLayout)),
	}
}

// EmailVerification is the email with the link that confirms an address.
func EmailVerification(to, name, link string, expires time.Time) Message {
	return Message{
		To:      to,
		Subject: "Confirm your Go Savvy email address",
		Text: fmt.Sprintf("Hello %s,\n\nOpen this link to confirm that this is your email address:\n\n%s\n\n"+
			"The link works once and expires on %s. If you did not expect this email, you can ignore it.\n",
			oneLine(name), link, expires.UTC().Format(expiryLayout)),
	}
}

// Invitation is the email with the link to join a space.
func Invitation(to, inviter, space, role, link string) Message {
	return Message{
		To:      to,
		Subject: "You are invited to " + oneLine(space) + " on Go Savvy",
		Text: fmt.Sprintf("%s invited you to the space \"%s\" on Go Savvy as %s.\n\n"+
			"Open this link to join:\n\n%s\n\nIf you did not expect this email, you can ignore it.\n",
			oneLine(inviter), oneLine(space), role, link),
	}
}

// oneLine keeps a name from breaking out of a header line.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
