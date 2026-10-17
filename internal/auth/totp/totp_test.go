package totp

import (
	"encoding/base32"
	"strings"
	"testing"
	"time"
)

// RFC 6238 appendix B: the SHA-1 secret "12345678901234567890" at T=59s is 94287082,
// whose last six digits are what an authenticator shows.
func TestTOTPMatchesTheRFCVector(t *testing.T) {
	key := []byte("12345678901234567890")
	if got := totpCode(key, 59/30); got != "287082" {
		t.Errorf("code at 59s = %s, want 287082", got)
	}
}

func TestVerifyStepAcceptsAdjacentSteps(t *testing.T) {
	secret, err := NewSecret()
	if err != nil {
		t.Fatal(err)
	}
	key, _ := decodeSecretForTest(secret)
	now := time.Unix(1_700_000_000, 0)
	cur := now.Unix() / 30

	for _, w := range []int64{-1, 0, 1} {
		step, ok := VerifyStep(secret, totpCode(key, cur+w), now)
		if !ok || step != cur+w {
			t.Errorf("code of step %+d: got step %d ok=%v, want %d", w, step, ok, cur+w)
		}
	}
	if _, ok := VerifyStep(strings.ToLower(secret), " "+totpCode(key, cur)+" ", now); !ok {
		t.Error("case of the secret and spaces around the code do not matter")
	}
	for _, w := range []int64{-5, -2, 2, 5} {
		if _, ok := VerifyStep(secret, totpCode(key, cur+w), now); ok {
			t.Errorf("a code %+d steps off verified", w)
		}
	}
	for _, bad := range []string{"", "12345", "1234567", "abcdef"} {
		if _, ok := VerifyStep(secret, bad, now); ok {
			t.Errorf("%q verified", bad)
		}
	}
	if _, ok := VerifyStep("!!!not base32!!!", "123456", now); ok {
		t.Error("a broken secret verified")
	}
}

func TestProvisioningURINamesTheAccount(t *testing.T) {
	uri := ProvisioningURI("ann@test.com", "ABC")
	for _, want := range []string{"otpauth://totp/", "secret=ABC", "issuer=Savvy", "digits=6", "period=30"} {
		if !strings.Contains(uri, want) {
			t.Errorf("%q missing from %s", want, uri)
		}
	}
}

func decodeSecretForTest(secret string) ([]byte, error) {
	return base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
}
