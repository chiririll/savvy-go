package httpserver

import (
	"savvy-go/internal/appconfig"
	"savvy-go/internal/auth"
	"savvy-go/internal/backup"
	"savvy-go/internal/ledger"
	"savvy-go/internal/locator"
	"savvy-go/internal/oauth"
	"savvy-go/internal/settings"
	"savvy-go/internal/signing"
	"savvy-go/internal/spaces"
	"savvy-go/internal/sso"
	"savvy-go/internal/store"
	"savvy-go/internal/transfers"
)

// services are the server-level services the tests set up and inspect directly, taken from the
// server's locator.
type services struct {
	Cfg            appconfig.Config
	Store          store.Store
	Keys           *signing.Holder
	Rates          ledger.RateSource
	Users          auth.Users
	Sessions       auth.Sessions
	APITokens      auth.APITokens
	PasswordTokens auth.PasswordTokens
	Challenges     auth.Challenges
	Settings       settings.Store
	Spaces         spaces.Service
	Backups        backup.Service
	Transfers      *transfers.Service
	SSO            *sso.Service
	OAuth          *oauth.Service
}

func servicesOf(s *Server) services {
	l := s.loc
	return services{
		Cfg: locator.Get[appconfig.Config](l), Store: locator.Get[store.Store](l),
		Keys: locator.Get[*signing.Holder](l), Rates: locator.Get[ledger.RateSource](l),
		Users: locator.Get[auth.Users](l), Sessions: locator.Get[auth.Sessions](l),
		APITokens: locator.Get[auth.APITokens](l), PasswordTokens: locator.Get[auth.PasswordTokens](l),
		Challenges: locator.Get[auth.Challenges](l), Settings: locator.Get[settings.Store](l),
		Spaces: locator.Get[spaces.Service](l), Backups: locator.Get[backup.Service](l),
		Transfers: locator.Get[*transfers.Service](l), SSO: locator.Get[*sso.Service](l),
		OAuth: locator.Get[*oauth.Service](l),
	}
}
