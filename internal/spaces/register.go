package spaces

import (
	"savvy-go/internal/locator"
	"savvy-go/internal/store"
)

// Register adds the space service. It needs the store.
func Register(l *locator.Locator) {
	locator.Provide(l, func(l *locator.Locator) (Service, error) {
		return Service{Store: locator.Get[store.Store](l)}, nil
	})
}
