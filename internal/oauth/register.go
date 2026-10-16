package oauth

import (
	"savvy-go/internal/locator"
	"savvy-go/internal/store"
)

// Register adds the OAuth service for MCP clients. It needs the server database.
func Register(l *locator.Locator) {
	locator.Provide(l, func(l *locator.Locator) (*Service, error) {
		return New(locator.Get[store.DB](l)), nil
	})
}
