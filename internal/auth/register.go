package auth

import (
	"savvy-go/internal/appconfig"
	"savvy-go/internal/config"
	"savvy-go/internal/locator"
	"savvy-go/internal/store"
)

// Register adds the services of this package to l. It needs the server database
// (store.DB), the app config and the config file (*config.Config) for [security].
func Register(l *locator.Locator) {
	locator.Provide(l, func(l *locator.Locator) (Users, error) {
		return Users{DB: locator.Get[store.DB](l)}, nil
	})
	locator.Provide(l, func(l *locator.Locator) (EmailVerifications, error) {
		return EmailVerifications{DB: locator.Get[store.DB](l)}, nil
	})
	locator.Provide(l, func(l *locator.Locator) (APITokens, error) {
		return APITokens{DB: locator.Get[store.DB](l)}, nil
	})
	locator.Provide(l, func(l *locator.Locator) (PasswordTokens, error) {
		return PasswordTokens{DB: locator.Get[store.DB](l)}, nil
	})
	locator.Provide(l, func(l *locator.Locator) (TwoFactor, error) {
		return TwoFactor{DB: locator.Get[store.DB](l), Users: locator.Get[Users](l)}, nil
	})
	locator.Provide(l, func(l *locator.Locator) (WebAuthn, error) {
		return WebAuthn{DB: locator.Get[store.DB](l), Cfg: locator.Get[appconfig.Config](l)}, nil
	})
	locator.Provide(l, func(l *locator.Locator) (Security, error) {
		return config.Of(locator.Get[*config.Config](l), SecuritySection).Get(), nil
	})
	locator.Provide(l, func(l *locator.Locator) (Sessions, error) {
		sec := locator.Get[Security](l)
		return Sessions{DB: locator.Get[store.DB](l), SessionTTL: sec.SessionTTL, RememberTTL: sec.RememberTTL}, nil
	})
	locator.Provide(l, func(l *locator.Locator) (Challenges, error) {
		return Challenges{DB: locator.Get[store.DB](l), TTL: locator.Get[Security](l).ChallengeTTL}, nil
	})
}
