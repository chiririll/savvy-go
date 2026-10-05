package signing

import (
	"errors"
	"testing"
)

// P50: a key that was in use is never silently replaced when its file is gone.
func TestP50MissingKeyIsNotRegenerated(t *testing.T) {
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
