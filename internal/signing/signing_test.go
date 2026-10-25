package signing

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"os"
	"testing"
)

// P50: a key that was in use is never silently replaced when its file is gone.
func TestP50MissingKeyNotRegenerated(t *testing.T) {
	dir := t.TempDir()
	if _, err := Load(dir, true); !errors.Is(err, ErrKeyMissing) {
		t.Fatalf("missing used key: %v", err)
	}
	first, err := Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Load(dir, true)
	if err != nil || again.KID() != first.KID() {
		t.Fatalf("reload %v %v", again, err)
	}
	msg := []byte("manifest")
	if !Verify(first.Public(), msg, again.Sign(msg)) || Verify(first.Public(), []byte("other"), again.Sign(msg)) {
		t.Fatal("signatures do not verify as expected")
	}
}

func TestKeyFileRoundTripMalformedKeys(t *testing.T) {
	dir := t.TempDir()
	key, err := Generate(dir)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(Path(dir))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(raw)
	if err != nil || parsed.KID() != key.KID() {
		t.Fatalf("a key file parses back to the same key: %v", err)
	}
	if len(key.KID()) != 16 || KID(key.Public()) != key.KID() {
		t.Errorf("kid %q", key.KID())
	}
	for name, bad := range map[string][]byte{"not base64": []byte("!!!"), "too short": []byte("AAAA"), "empty": nil} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if err := Write(dir, []byte("short")); err == nil {
		t.Error("a seed of the wrong size was written")
	}

	// A server backup carries the seed; restoring it brings the same identity back.
	other := t.TempDir()
	if err := Write(other, key.Seed()); err != nil {
		t.Fatal(err)
	}
	restored, err := Load(other, true)
	if err != nil || restored.KID() != key.KID() {
		t.Errorf("restore: %v", err)
	}

	again, err := Generate(dir)
	if err != nil || again.KID() == key.KID() {
		t.Errorf("Generate replaces the key: %v", err)
	}
	if err := os.WriteFile(Path(dir), []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir, true); err == nil || errors.Is(err, ErrKeyMissing) {
		t.Errorf("a damaged key file is an error of its own: %v", err)
	}
}

func TestVerifyRefusesWrongKeysSignatures(t *testing.T) {
	a, _ := Generate(t.TempDir())
	b, _ := Generate(t.TempDir())
	msg := []byte("m")
	sig := a.Sign(msg)
	if !Verify(a.Public(), msg, sig) || Verify(b.Public(), msg, sig) || Verify(a.Public()[:5], msg, sig) ||
		Verify(a.Public(), msg, sig[:10]) ||
		Verify(nil, msg, sig) {
		t.Error("Verify")
	}
}

func TestKeyRingSaysWhoSignedWhat(t *testing.T) {
	own, _ := Generate(t.TempDir())
	friend, _ := Generate(t.TempDir())
	stranger, _ := Generate(t.TempDir())
	ring := KeyRing{
		Holder: NewHolder(own),
		Trusted: func(kid string) ed25519.PublicKey {
			if kid == friend.KID() {
				return friend.Public()
			}
			return nil
		},
	}
	msg := []byte("manifest")
	enc := func(k *Key, m []byte) []byte { return []byte(base64.StdEncoding.EncodeToString(k.Sign(m))) }

	tests := []struct {
		name string
		kid  string
		msg  []byte
		sig  []byte
		want Signature
	}{
		{"this server", own.KID(), msg, enc(own, msg), SignedHere},
		{"a trusted server", friend.KID(), msg, enc(friend, msg), SignedTrusted},
		{"a stranger", stranger.KID(), msg, enc(stranger, msg), Unsigned},
		{"a trusted kid with another key's signature", friend.KID(), msg, enc(stranger, msg), Unsigned},
		{"our kid claimed by a stranger's signature", own.KID(), msg, enc(stranger, msg), Unsigned},
		{"a changed document", own.KID(), []byte("changed"), enc(own, msg), Unsigned},
		{"no signature", own.KID(), msg, nil, Unsigned},
		{"a signature that is not base64", own.KID(), msg, []byte("!!"), Unsigned},
		{"surrounding spaces are fine", own.KID(), msg, []byte(" " + string(enc(own, msg)) + "\n"), SignedHere},
	}
	for _, tc := range tests {
		if got := ring.Check(tc.kid, tc.msg, tc.sig); got != tc.want {
			t.Errorf("%s: %s, want %s", tc.name, got, tc.want)
		}
	}
	if got := (KeyRing{}).Check(own.KID(), msg, enc(own, msg)); got != Unsigned {
		t.Errorf("a ring without a holder trusts nothing: %s", got)
	}
}

func TestHolderSwapsKeyForEveryReader(t *testing.T) {
	a, _ := Generate(t.TempDir())
	b, _ := Generate(t.TempDir())
	h := NewHolder(a)
	ring := KeyRing{Holder: h}
	sig := []byte(base64.StdEncoding.EncodeToString(a.Sign([]byte("m"))))
	if ring.Check(a.KID(), []byte("m"), sig) != SignedHere {
		t.Fatal("before the restore")
	}
	h.Set(b)
	if h.Key() != b || ring.Check(a.KID(), []byte("m"), sig) != Unsigned {
		t.Error("after a server restore the old key no longer counts as our own")
	}
}
