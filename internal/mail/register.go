package mail

import (
	"savvy-go/internal/appconfig"
	"savvy-go/internal/auth"
	"savvy-go/internal/config"
	"savvy-go/internal/locator"
	"savvy-go/internal/mail/smtp"
)

// Register adds the mail service and the email verifier. It needs the config file
// (*config.Config) and the shared config. A smtp.Mailer in the locator replaces the SMTP
// sender of the [mailer] section; tests use it to record mail.
func Register(l *locator.Locator) {
	locator.Provide(l, func(l *locator.Locator) (*Service, error) {
		sender, _ := locator.TryGet[smtp.Mailer](l) // optional
		return New(locator.Get[*config.Config](l), locator.Get[appconfig.Config](l), sender), nil
	})
	locator.Provide(l, func(l *locator.Locator) (Verifier, error) {
		return Verifier{
			Cfg:           locator.Get[appconfig.Config](l),
			Mail:          locator.Get[*Service](l),
			Verifications: locator.Get[auth.EmailVerifications](l),
			Users:         locator.Get[auth.Users](l),
		}, nil
	})
}
