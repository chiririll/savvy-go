package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func TestLaravelEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	appKey := "base64:" + base64.StdEncoding.EncodeToString(key)
	plain := "JBSWY3DPEHPK3PXP"
	ct, err := EncryptLaravel(appKey, plain)
	if err != nil {
		t.Fatal(err)
	}
	if !LooksLikeLaravelEncrypted(ct) {
		t.Fatalf("envelope %s", ct)
	}
	got, err := DecryptLaravel(appKey, ct)
	if err != nil || got != plain {
		t.Fatalf("decrypt %q %v", got, err)
	}
}

func TestLaravelDecryptRejectsBadMAC(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	appKey := "base64:" + base64.StdEncoding.EncodeToString(key)
	ct, err := EncryptLaravel(appKey, "ABCDEFGH")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := base64.StdEncoding.DecodeString(ct)
	raw[len(raw)/2] ^= 0xff
	bad := base64.StdEncoding.EncodeToString(raw)
	if _, err := DecryptLaravel(appKey, bad); err == nil {
		t.Fatal("expected mac failure")
	}
}

func TestUnwrapLegacyTOTPSecrets(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	appKey := "base64:" + base64.StdEncoding.EncodeToString(key)
	plain := "MFRGGZDFMZTWQ2LK"
	ct, err := EncryptLaravel(appKey, plain)
	if err != nil {
		t.Fatal(err)
	}

	// Minimal sqlite via the httpserver tests is heavier; exercise UnwrapSecret + persist helper.
	got, ok := UnwrapSecret(appKey, ct)
	if !ok || got != plain {
		t.Fatalf("unwrap %q %v", got, ok)
	}
	if _, ok := UnwrapSecret("", ct); ok {
		t.Fatal("must not unwrap without APP_KEY")
	}
	if s, ok := UnwrapSecret(appKey, plain); ok || s != plain {
		t.Fatalf("plaintext pass-through %q %v", s, ok)
	}
}

func TestVerifyTOTPAfterLaravelUnwrap(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	appKey := "base64:" + base64.StdEncoding.EncodeToString(key)
	secret, err := NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	ct, err := EncryptLaravel(appKey, secret)
	if err != nil {
		t.Fatal(err)
	}
	tf := TwoFactor{AppKey: appKey}
	u := &User{TwoFactorSecret: &ct, TwoFactorEnabled: true, TwoFactorConfirmed: true}
	if !tf.verifyTOTP(context.Background(), u, TOTPNow(secret)) {
		t.Fatal("legacy encrypted secret should verify")
	}
}
