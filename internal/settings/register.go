package settings

import (
	"savvy-go/internal/locator"
	"savvy-go/internal/store"
)

// Register adds the settings store, which keeps the server-level settings in the server
// database (store.DB).
func Register(l *locator.Locator) {
	locator.Provide(l, func(l *locator.Locator) (Store, error) {
		return Store{DB: locator.Get[store.DB](l)}, nil
	})
}
