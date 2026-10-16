package mail

import (
	"context"
	"errors"

	"savvy-go/internal/auth"
	"savvy-go/internal/mail/smtp"
)

// ErrOwnAddressUnverified is returned when an admin would require verified
// addresses while their own is not confirmed: they would lock themselves out.
var ErrOwnAddressUnverified = errors.New("confirm your own email address first, or you would lock yourself out")

// MailRequiredError says why confirmation links cannot be sent.
type MailRequiredError struct{ Reason string }

func (e MailRequiredError) Error() string {
	return "Email verification needs working mail and a public URL. " + e.Reason
}

// Verifier sends confirmation links and confirms addresses.
type Verifier struct {
	AppURL        func() string // the public URL of the instance, empty when unset
	Mail          *Service
	Verifications auth.EmailVerifications
	Users         auth.Users
}

// CanRequire reports why the requirement cannot be switched on by admin: links
// need mail and a public URL, and the admin must have confirmed their own address.
func (s Verifier) CanRequire(admin *auth.User) error {
	if !s.Mail.Mailer().Enabled() || s.AppURL() == "" {
		reason := s.Mail.OffReason()
		if s.Mail.Mailer().Enabled() {
			reason = NoAppURLMessage
		}
		return MailRequiredError{Reason: reason}
	}
	if !admin.EmailVerified {
		return ErrOwnAddressUnverified
	}
	return nil
}

// Send emails u a link that confirms their address. It says what
// is missing when mail cannot be sent.
func (s Verifier) Send(ctx context.Context, u *auth.User) string {
	switch {
	case !s.Mail.Mailer().Enabled():
		return s.Mail.OffReason()
	case s.AppURL() == "":
		return NoAppURLMessage
	}
	token, exp, err := s.Verifications.Issue(ctx, u)
	if err != nil {
		return "Could not create the link: " + err.Error()
	}
	if err := s.Mail.Send(ctx, smtp.EmailVerification(u.Email, u.Name, s.AppURL()+"/verify-email/"+token, exp)); err != nil {
		return "Could not send the email: " + err.Error()
	}
	return ""
}

// Confirm marks the address named by the emailed token as confirmed. It reports
// false for a token that is invalid or has expired.
func (s Verifier) Confirm(ctx context.Context, token string) (bool, error) {
	u, err := s.Verifications.Consume(ctx, token)
	if err != nil || u == nil {
		return false, err
	}
	return true, s.Users.SetEmailVerified(ctx, u.ID, true)
}
