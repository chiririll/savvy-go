package transfers

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"

	repo "savvy-go/internal/db/repo/transfers"
	"savvy-go/internal/signing"
	"savvy-go/internal/spacekeys"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
)

// transferPayloadV1 is what sig_v 1 and 2 sign. A later format adds fields in
// a new struct and a new sig_v; records keep verifying by the format they were
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

// sig_v 1: the server key signs (v1.0, until re-signed). sig_v 2: the key of
// each of the two spaces signs the same payload, the source's in source_*, the
// destination's in destination_*.
const currentSigV = 2

// payload is what sigV signs: the stored bytes when the record was read from
// the database or signed here, otherwise the format rebuilt from the fields.
func (t Transfer) payload(sigV int64) ([]byte, bool) {
	switch sigV {
	case 1, 2:
		if len(t.signed) > 0 && t.SigV == sigV {
			return t.signed, true
		}
		b, err := json.Marshal(
			transferPayloadV1{t.UUID, t.From, t.To, t.Date, t.Description, t.CreatedBy, t.Version, t.DeletedAt},
		)
		return b, err == nil
	}
	return nil, false
}

// signWithSpaces signs t with the current keys of its two spaces, made on the
// spot when a space has none. dbs holds the handle of each space, and so the
// transaction the keys are written in.
func (s Service) signWithSpaces(ctx context.Context, dbs map[int64]store.DB, t *Transfer) error {
	t.signed = nil // the fields changed; sign what they now say
	return s.signStored(ctx, dbs, t)
}

// signStored signs t with sig_v 2 over t.signed when set (re-signing a record
// keeps the bytes it was first signed with), else over the format rebuilt
// from its fields.
func (s Service) signStored(ctx context.Context, dbs map[int64]store.DB, t *Transfer) error {
	_, byUUID, err := s.spaceUUIDs(ctx)
	if err != nil {
		return err
	}
	handle := func(uuid string) (store.DB, error) {
		sp, ok := byUUID[uuid]
		if !ok || dbs[sp.ID] == nil {
			return nil, ErrSpaceMissing
		}
		return dbs[sp.ID], nil
	}
	src, err := handle(t.From.SpaceUUID)
	if err != nil {
		return err
	}
	dst, err := handle(t.To.SpaceUUID)
	if err != nil {
		return err
	}
	srcKey, err := spacekeys.Current(ctx, src)
	if err != nil {
		return err
	}
	dstKey, err := spacekeys.Current(ctx, dst)
	if err != nil {
		return err
	}
	t.SigV = currentSigV
	p, _ := t.payload(currentSigV)
	t.signed = p
	t.SignerKID, t.Signature = srcKey.KID(), base64.StdEncoding.EncodeToString(srcKey.Sign(p))
	t.DestKID, t.DestSignature = dstKey.KID(), base64.StdEncoding.EncodeToString(dstKey.Sign(p))
	return nil
}

// verifier checks signatures. A sig_v 2 signature is checked against the keys
// in the file of its own space, read from the live database on this server, so
// a transfer is valid only while both of its spaces are here.
type verifier struct {
	s      Service
	ctx    context.Context
	dbs    map[int64]store.DB
	byUUID map[string]spaces.Space
	keys   map[string]map[string]ed25519.PublicKey // by space uuid, then kid
}

// verifier prepares checks inside an InSpaces call (dbs is its handles) or,
// with nil, outside one.
func (s Service) verifier(ctx context.Context, dbs map[int64]store.DB) (*verifier, error) {
	_, byUUID, err := s.spaceUUIDs(ctx)
	if err != nil {
		return nil, err
	}
	return &verifier{s: s, ctx: ctx, dbs: dbs, byUUID: byUUID, keys: map[string]map[string]ed25519.PublicKey{}}, nil
}

// spaceKeys are the public keys of a space on this server, nil when it is not.
func (v *verifier) spaceKeys(uuid string) map[string]ed25519.PublicKey {
	if k, ok := v.keys[uuid]; ok {
		return k
	}
	var found map[string]ed25519.PublicKey
	if sp, ok := v.byUUID[uuid]; ok {
		d := v.dbs[sp.ID]
		if d == nil {
			d, _ = v.s.st().Space(v.ctx, sp.ID)
		}
		if d != nil {
			found, _ = spacekeys.Public(v.ctx, d)
		}
	}
	v.keys[uuid] = found
	return found
}

// ok reports whether t is signed as its sig_v says: by this server or a key it
// trusts (1), or by both of its spaces (2).
func (v *verifier) ok(t Transfer) bool {
	p, ok := t.payload(t.SigV)
	if !ok {
		return false
	}
	switch t.SigV {
	case 1:
		return v.s.Keys.Check(t.SignerKID, p, []byte(t.Signature)) != signing.Unsigned
	case 2:
		return v.signedBy(t.From.SpaceUUID, t.SignerKID, t.Signature, p) &&
			v.signedBy(t.To.SpaceUUID, t.DestKID, t.DestSignature, p)
	}
	return false
}

func (v *verifier) signedBy(spaceUUID, kid, sig string, payload []byte) bool {
	pub := v.spaceKeys(spaceUUID)[kid]
	raw, err := base64.StdEncoding.DecodeString(sig)
	return pub != nil && err == nil && signing.Verify(pub, payload, raw)
}

// verify checks one transfer outside an InSpaces call.
func (s Service) verify(ctx context.Context, t Transfer) bool {
	v, err := s.verifier(ctx, nil)
	return err == nil && v.ok(t)
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
