package auth

import (
	"savvy-go/internal/appconfig"

	"errors"
	"testing"
)

func TestWebAuthnEngineUsesTheOriginOfThePublicURL(t *testing.T) {
	w := WebAuthn{Cfg: appconfig.Config{}.WithAppURL("https://savvy.example.com:8443/app")}
	wa, err := w.engine()
	if err != nil {
		t.Fatal(err)
	}
	if wa.Config.RPID != "savvy.example.com" {
		t.Errorf("RPID = %q", wa.Config.RPID)
	}
	if got := wa.Config.RPOrigins; len(got) != 1 || got[0] != "https://savvy.example.com:8443" {
		t.Errorf("origins = %v, want the origin without the path", got)
	}
}

func TestWebAuthnEngineNeedsAPublicURL(t *testing.T) {
	for _, url := range []string{"", "not a url", "/relative"} {
		if _, err := (WebAuthn{Cfg: appconfig.Config{}.WithAppURL(url)}).engine(); !errors.Is(err, ErrNoPublicURL) {
			t.Errorf("engine(%q) = %v, want ErrNoPublicURL", url, err)
		}
	}
}
