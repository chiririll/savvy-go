package sso

import (
	"savvy-go/internal/appconfig"
	"savvy-go/internal/auth"
	"savvy-go/internal/locator"
	"savvy-go/internal/settings"
	"savvy-go/internal/store"
)

// Register adds the sign-in service. It needs the services of auth and settings.
func Register(l *locator.Locator) {
	locator.Provide(l, func(l *locator.Locator) (*Service, error) {
		return &Service{
			DB:       locator.Get[store.DB](l),
			Users:    locator.Get[auth.Users](l),
			Settings: locator.Get[settings.Store](l),
			Cfg:      locator.Get[appconfig.Config](l),
		}, nil
	})
}
