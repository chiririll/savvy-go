package emaillink_test

import (
	"strings"
	"testing"
	"time"

	"savvy-go/internal/emaillink"
	"savvy-go/internal/signing"
)

func newSigner(t *testing.T) emaillink.Signer {
	t.Helper()
	key, err := signing.Load(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	return emaillink.Signer{Keys: signing.NewHolder(key)}
}

func TestVerificationTokenIsBoundToUserAddressAndTime(t *testing.T) {
	s := newSigner(t)
	now := time.Now()
	tok := s.Verification(7, "Ann@Test.com", now.Add(time.Hour))

	id, ok := s.VerificationUser(tok)
	if !ok || id != 7 {
		t.Fatalf("user = %d, %v", id, ok)
	}
	if !s.VerificationValid(tok, "ann@test.com", now) {
		t.Fatal("valid token refused (the address is compared without case)")
	}
	if s.VerificationValid(tok, "other@test.com", now) {
		t.Fatal("a token must stop working once the address changed")
	}
	if s.VerificationValid(tok, "ann@test.com", now.Add(2*time.Hour)) {
		t.Fatal("an expired token was accepted")
	}

	// Changing the user or the expiry in the text breaks the signature.
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("token %q", tok)
	}
	forged := "8." + parts[1] + "." + parts[2]
	if s.VerificationValid(forged, "ann@test.com", now) {
		t.Fatal("a token for another user was accepted")
	}
	for _, bad := range []string{"", "x", "1.2", "a.b.c", tok + "x", tok[:len(tok)-2]} {
		if s.VerificationValid(bad, "ann@test.com", now) {
			t.Fatalf("%q was accepted", bad)
		}
	}
}

func TestInviteSignatureProvesTheLinkWasEmailed(t *testing.T) {
	s := newSigner(t)
	sig := s.Invite("invite-token", "New@Test.com")
	if sig == "" {
		t.Fatal("no signature")
	}
	if !s.InviteValid("invite-token", "new@test.com", sig) {
		t.Fatal("valid signature refused (the address is compared without case)")
	}
	for _, c := range []struct{ token, email, sig string }{
		{"invite-token", "new@test.com", ""},
		{"invite-token", "other@test.com", sig},
		{"other-token", "new@test.com", sig},
		{"invite-token", "new@test.com", sig + "A"},
		{"invite-token", "new@test.com", "not base64 !"},
	} {
		if s.InviteValid(c.token, c.email, c.sig) {
			t.Fatalf("accepted %+v", c)
		}
	}
}

func TestPurposesDoNotMix(t *testing.T) {
	s := newSigner(t)
	now := time.Now()
	tok := s.Verification(1, "a@test.com", now.Add(time.Hour))
	sig := tok[strings.LastIndex(tok, ".")+1:]
	// A verification signature is not an invitation signature, whatever it signs.
	if s.InviteValid("1", "a@test.com", sig) {
		t.Fatal("a signature crossed purposes")
	}
}

func TestSignaturesDieWithTheKey(t *testing.T) {
	a, b := newSigner(t), newSigner(t)
	if b.InviteValid("t", "a@test.com", a.Invite("t", "a@test.com")) {
		t.Fatal("another server's signature was accepted")
	}
}
