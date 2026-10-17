package uploads

import (
	"crypto/rand"
	"encoding/hex"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/locator"
	"savvy-go/internal/store"
)

// Register adds the upload service. It needs the server database and the shared config.
func Register(l *locator.Locator) {
	locator.Provide(l, func(l *locator.Locator) (Service, error) {
		secret, err := randomSecret()
		if err != nil {
			return Service{}, err
		}
		return Service{
			DB:         locator.Get[store.DB](l),
			Root:       locator.Get[appconfig.Config](l).UploadsDir,
			SignSecret: secret,
		}, nil
	})
}

// randomSecret is a fresh secret for signing upload part URLs. It lives as long as the
// process: those URLs expire within minutes, and an upload interrupted by a restart starts
// over.
func randomSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
