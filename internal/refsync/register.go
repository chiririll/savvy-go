package refsync

import (
	"savvy-go/internal/locator"
	"savvy-go/internal/spaces"
)

// Register adds the reference-data sync service. It needs the space service.
func Register(l *locator.Locator) {
	locator.Provide(l, func(l *locator.Locator) (Service, error) {
		return Service{Spaces: locator.Get[spaces.Service](l)}, nil
	})
}
