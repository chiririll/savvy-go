package transfers

import (
	"savvy-go/internal/locator"
	"savvy-go/internal/signing"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
)

// Register adds the key ring (the instance's own key plus the public keys other instances
// are trusted with) and the transfer service. It needs the server database, the space
// service and the instance's *signing.Holder.
func Register(l *locator.Locator) {
	locator.Provide(l, func(l *locator.Locator) (signing.KeyRing, error) {
		return signing.KeyRing{
			Holder:  locator.Get[*signing.Holder](l),
			Trusted: TrustedKeys(locator.Get[store.DB](l)),
		}, nil
	})
	locator.Provide(l, func(l *locator.Locator) (*Service, error) {
		return &Service{Spaces: locator.Get[spaces.Service](l), Keys: locator.Get[signing.KeyRing](l)}, nil
	})
}
