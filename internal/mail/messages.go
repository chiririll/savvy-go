package mail

import (
	"fmt"
	"strings"
	"time"

	"savvy-go/internal/mail/smtp"
)

// The emails Go Savvy sends. Each takes plain values and returns an smtp.Message; the
// caller builds the link, so a message never decides what a link proves. The first
// argument is the language of the text (see Lang); the HTML layout is in layout/, the content of each message in messages/ and the texts in locales/.

// PasswordReset is the email with a one-time link to choose a new password.
func PasswordReset(lang, to, name, link string, expires time.Time) smtp.Message {
	return render(lang, "password_reset", to, link, fields{Name: name, Expires: expires})
}

// EmailVerification is the email with the link that confirms an address.
func EmailVerification(lang, to, name, link string, expires time.Time) smtp.Message {
	return render(lang, "email_verification", to, link, fields{Name: name, Expires: expires})
}

// Invitation is the email with the link to join a space.
func Invitation(lang, to, inviter, space, role, link string) smtp.Message {
	return render(lang, "invitation", to, link, fields{Inviter: inviter, Space: space, Role: role})
}

// Test is the email an admin sends to try the mail settings. It has no link and no button.
func Test(lang, to string) smtp.Message {
	return render(lang, "test", to, "", fields{})
}

// fields are the values a message puts into its text.
type fields struct {
	Name, Inviter, Space, Role string
	Expires                    time.Time
}

// oneLine keeps a name from breaking out of a header line.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// date is a day and month in the language of c: "4 March". No year and no time, since a link lives for days.
func (c catalog) date(t time.Time) string {
	t = t.UTC()
	return fmt.Sprintf("%d %s", t.Day(), c.Months[t.Month()-1])
}
