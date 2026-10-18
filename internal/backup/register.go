package backup

import (
	"savvy-go/internal/appconfig"
	"savvy-go/internal/locator"
	"savvy-go/internal/signing"
	"savvy-go/internal/store"
	"savvy-go/internal/transfers"
)

// Register adds the backup service. It needs the store, the key ring and transfers, and
// the app config and build for where backups go and which release made them.
func Register(l *locator.Locator) {
	locator.Provide(l, func(l *locator.Locator) (Service, error) {
		cfg := locator.Get[appconfig.Config](l)
		return Service{
			Store:      locator.Get[store.Store](l),
			Keys:       locator.Get[signing.KeyRing](l),
			Dir:        cfg.BackupsDir,
			DataDir:    cfg.DataDir,
			Transfers:  locator.Get[*transfers.Service](l),
			AppVersion: locator.Get[appconfig.Build](l).Version,
		}, nil
	})
}
