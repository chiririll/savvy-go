package signin

import (
	"savvy-go/internal/locator"
	"savvy-go/internal/settings"
	"savvy-go/internal/sso"
)

// Register adds the sign-in policy. It needs the settings store and the SSO service.
func Register(l *locator.Locator) {
	locator.Provide(l, func(l *locator.Locator) (Policy, error) {
		return Policy{Settings: locator.Get[settings.Store](l), SSO: locator.Get[*sso.Service](l)}, nil
	})
}
