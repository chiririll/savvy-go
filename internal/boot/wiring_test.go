package boot

import (
	"testing"

	"savvy-go/internal/app"
	"savvy-go/internal/appconfig"
	"savvy-go/internal/locator"
)

// app.New builds every registered provider, so a missing dependency fails the start instead
// of the first request that needs it.
func TestEveryRegisteredServiceResolves(t *testing.T) {
	st, cfg := openStore(t)
	defer st.Close()
	loc, err := app.New(cfg, st, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if locator.Get[appconfig.Config](loc).DataDir != cfg.DataDir {
		t.Error("the shared config must be in the locator")
	}
}
