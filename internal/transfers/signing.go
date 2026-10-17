package transfers

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"

	repo "savvy-go/internal/db/repo/transfers"
	"savvy-go/internal/signing"
	"savvy-go/internal/store"
)

// transferPayloadV1 is what sig_v 1 signs. A later format adds fields in a
// new struct and a new sig_v; records keep verifying by the format they were
// signed with (P49).
type transferPayloadV1 struct {
	UUID        string  `json:"uuid"`
	From        Side    `json:"from"`
	To          Side    `json:"to"`
	Date        string  `json:"date"`
	Description *string `json:"description"`
	CreatedBy   *int64  `json:"created_by"`
	Version     int64   `json:"version"`
	DeletedAt   *string `json:"deleted_at"`
}

const currentSigV = 1

func (t Transfer) payload(sigV int64) ([]byte, bool) {
	switch sigV {
	case 1:
		b, err := json.Marshal(
			transferPayloadV1{t.UUID, t.From, t.To, t.Date, t.Description, t.CreatedBy, t.Version, t.DeletedAt},
		)
		return b, err == nil
	}
	return nil, false
}

func signTransfer(k signing.KeyRing, t *Transfer) {
	key := k.Holder.Key()
	t.SigV = currentSigV
	p, _ := t.payload(currentSigV)
	t.SignerKID = key.KID()
	t.Signature = base64.StdEncoding.EncodeToString(key.Sign(p))
}

// verify reports whether t is signed by this server or a key it trusts.
func verifyTransfer(k signing.KeyRing, t Transfer) bool {
	p, ok := t.payload(t.SigV)
	if !ok {
		return false
	}
	return k.Check(t.SignerKID, p, []byte(t.Signature)) != signing.Unsigned
}

// TrustedKeys looks up a trusted public key in the server database.
func TrustedKeys(server store.DB) func(kid string) ed25519.PublicKey {
	return func(kid string) ed25519.PublicKey {
		raw, ok := (repo.Server{DB: server}).PublicKey(context.Background(), kid)
		if !ok {
			return nil
		}
		pub, err := base64.StdEncoding.DecodeString(raw)
		if err != nil || len(pub) != ed25519.PublicKeySize {
			return nil
		}
		return pub
	}
}
