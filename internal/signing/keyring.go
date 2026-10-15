package signing

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"strings"
)

// Signature is how far a signed document can be trusted.
type Signature string

const (
	SignedHere    Signature = "own"      // signed by this server's key
	SignedTrusted Signature = "trusted"  // signed by a key this server trusts
	Unsigned      Signature = "unsigned" // no or an unknown signature
)

// KeyRing checks signatures: this server's key and the keys it trusts.
type KeyRing struct {
	Holder  *Holder
	Trusted func(kid string) ed25519.PublicKey
}

// Check reports who, if anyone, signed msg with the base64 signature sig
// made by the key kid.
func (k KeyRing) Check(kid string, msg, sig []byte) Signature {
	raw, err := decodeSig(sig)
	if err != nil || k.Holder == nil {
		return Unsigned
	}
	if own := k.Holder.Key(); own != nil && own.KID() == kid && Verify(own.Public(), msg, raw) {
		return SignedHere
	}
	if k.Trusted != nil {
		if pub := k.Trusted(kid); pub != nil && Verify(pub, msg, raw) {
			return SignedTrusted
		}
	}
	return Unsigned
}

func decodeSig(b []byte) ([]byte, error) {
	if len(b) == 0 {
		return nil, errors.New("no signature")
	}
	return base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
}
