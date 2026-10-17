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

func TestVerifyTOTPAcceptsOneStepEitherWay(t *testing.T) {
	secret, err := NewSecret()
	if err != nil {
		t.Fatal(err)
	}
	if !Verify(secret, Now(secret)) {
		t.Fatal("the current code must verify")
	}
	if !Verify(strings.ToLower(secret), " "+Now(secret)+" ") {
		t.Error("case of the secret and spaces around the code do not matter")
	}
	key, _ := decodeSecretForTest(secret)
	now := time.Now().Unix() / 30
	if !Verify(secret, totpCode(key, now-1)) || !Verify(secret, totpCode(key, now+1)) {
		t.Error("a code one step off must still verify, for clock drift")
	}
	if Verify(secret, totpCode(key, now-5)) {
		t.Error("an old code verified")
	}
	for _, bad := range []string{"", "12345", "1234567", "abcdef"} {
		if Verify(secret, bad) {
			t.Errorf("%q verified", bad)
		}
	}
	if Verify("!!!not base32!!!", "123456") {
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
