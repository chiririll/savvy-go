package auth

import (
	"encoding/base32"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordHashesCheckOut(t *testing.T) {
	h, err := HashPassword("s3cret")
	if err != nil {
		t.Fatal(err)
	}
	if h == "s3cret" || !CheckPassword(h, "s3cret") {
		t.Fatal("a hash must verify its own password and not be the password")
	}
	if CheckPassword(h, "S3cret") || CheckPassword(h, "") || CheckPassword("", "s3cret") || CheckPassword("not a hash", "s3cret") {
		t.Error("a wrong password, an empty one or a broken hash verified")
	}
	other, _ := HashPassword("s3cret")
	if other == h {
		t.Error("hashes of one password must differ (salted)")
	}
}

// The Laravel app wrote $2y$ hashes; users who came from it still sign in.
func TestLaravelPasswordHashesStillVerify(t *testing.T) {
	b, err := bcrypt.GenerateFromPassword([]byte("old-pass"), 4)
	if err != nil {
		t.Fatal(err)
	}
	laravel := "$2y$" + string(b)[4:]
	if !CheckPassword(laravel, "old-pass") || CheckPassword(laravel, "wrong") {
		t.Error("a $2y$ hash must verify like a $2a$ one")
	}
}

func TestRandomStringAndTokenHash(t *testing.T) {
	a, b := RandomString(32), RandomString(32)
	if len(a) != 32 || a == b || strings.Trim(a, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789") != "" {
		t.Errorf("random strings %q %q", a, b)
	}
	if HashToken("x") != HashToken("x") || HashToken("x") == HashToken("y") || len(HashToken("x")) != 64 {
		t.Error("token hashes are stable, distinct and hex sha-256")
	}
}

// RFC 6238 appendix B: the SHA-1 secret "12345678901234567890" at T=59s is 94287082,
// whose last six digits are what an authenticator shows.
func TestTOTPMatchesTheRFCVector(t *testing.T) {
	key := []byte("12345678901234567890")
	if got := totpCode(key, 59/30); got != "287082" {
		t.Errorf("code at 59s = %s, want 287082", got)
	}
}

func TestVerifyTOTPAcceptsOneStepEitherWay(t *testing.T) {
	secret, err := NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyTOTP(secret, TOTPNow(secret)) {
		t.Fatal("the current code must verify")
	}
	if !VerifyTOTP(strings.ToLower(secret), " "+TOTPNow(secret)+" ") {
		t.Error("case of the secret and spaces around the code do not matter")
	}
	key, _ := decodeSecretForTest(secret)
	now := time.Now().Unix() / 30
	if !VerifyTOTP(secret, totpCode(key, now-1)) || !VerifyTOTP(secret, totpCode(key, now+1)) {
		t.Error("a code one step off must still verify, for clock drift")
	}
	if VerifyTOTP(secret, totpCode(key, now-5)) {
		t.Error("an old code verified")
	}
	for _, bad := range []string{"", "12345", "1234567", "abcdef"} {
		if VerifyTOTP(secret, bad) {
			t.Errorf("%q verified", bad)
		}
	}
	if VerifyTOTP("!!!not base32!!!", "123456") {
		t.Error("a broken secret verified")
	}
}

func decodeSecretForTest(secret string) ([]byte, error) {
	return base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
}

func TestProvisioningURINamesTheAccount(t *testing.T) {
	uri := TOTPProvisioningURI("ann@test.com", "ABC")
	for _, want := range []string{"otpauth://totp/", "secret=ABC", "issuer=Savvy", "digits=6", "period=30"} {
		if !strings.Contains(uri, want) {
			t.Errorf("%q missing from %s", want, uri)
		}
	}
}

func TestRecoveryCodesLookRight(t *testing.T) {
	seen := map[string]bool{}
	for range 200 {
		c := recoveryCode()
		if len(c) != 9 || c[4] != '-' {
			t.Fatalf("code %q", c)
		}
		if strings.ContainsAny(c, "ilo01") {
			t.Fatalf("%q has characters that are easy to misread", c)
		}
		seen[c] = true
	}
	if len(seen) < 190 {
		t.Errorf("only %d distinct codes in 200", len(seen))
	}
}
