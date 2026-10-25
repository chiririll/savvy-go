package app

import (
	"savvy-go/internal/app/signin"
	"savvy-go/internal/auth"
	"savvy-go/internal/backup"
	"savvy-go/internal/locator"
	"savvy-go/internal/mail"
	"savvy-go/internal/oauth"
	"savvy-go/internal/refsync"
	"savvy-go/internal/settings"
	"savvy-go/internal/spaces"
	"savvy-go/internal/sso"
	"savvy-go/internal/transfers"
	"savvy-go/internal/uploads"
)

// registerServices has every service package add its constructors to the locator.
func registerServices(l *locator.Locator) {
	for _, register := range []func(*locator.Locator){
		auth.Register, settings.Register, spaces.Register, refsync.Register, transfers.Register,
		backup.Register, uploads.Register, sso.Register, oauth.Register, mail.Register, signin.Register,
	} {
		register(l)
	}
}
