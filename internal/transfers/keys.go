package transfers

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/settings"
	"savvy-go/internal/signing"
)

// TrustedKey is a public key this server accepts signatures of.
type TrustedKey struct {
	KID       string
	PublicKey string
	Name      string
	CreatedAt *time.Time
}

// TrustedKeysList lists the trusted keys.
func (s Service) TrustedKeysList(ctx context.Context) ([]TrustedKey, error) {
	rows, err := s.server().TrustedKeys(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]TrustedKey, len(rows))
	for i, r := range rows {
		out[i] = TrustedKey{KID: r.KID, PublicKey: r.PublicKey, Name: r.Name, CreatedAt: r.CreatedAt}
	}
	return out, nil
}

// Trust adds a public key (base64) to the trusted keys.
func (s Service) TrustKey(ctx context.Context, actor *auth.User, name, publicKey string) (string, error) {
	pub, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return "", errors.New("the public key must be a base64 Ed25519 key")
	}
	kid := signing.KID(pub)
	if err := s.server().InsertTrustedKey(ctx, kid, publicKey, name, actor.ID, time.Now().UTC()); err != nil {
		return "", err
	}
	return kid, s.Spaces.Audit(ctx, actor, "trust_key", nil, nil, kid+" "+name)
}

// UntrustKey removes a trusted key.
func (s Service) UntrustKey(ctx context.Context, actor *auth.User, kid string) error {
	ok, err := s.server().DeleteTrustedKey(ctx, kid)
	if err != nil {
		return err
	}
	if !ok {
		return ErrGone
	}
	return s.Spaces.Audit(ctx, actor, "untrust_key", nil, nil, kid)
}

// RotateKey replaces the server key; the old public key becomes trusted, so
// what it signed still verifies (P47).
func (s Service) RotateKey(ctx context.Context, actor *auth.User, dataDir string) (string, error) {
	old := s.Keys.Holder.Key()
	if old != nil {
		pub := base64.StdEncoding.EncodeToString(old.Public())
		if err := s.server().InsertTrustedKey(ctx, old.KID(), pub, "previous server key", actor.ID,
			time.Now().UTC()); err != nil {
			return "", err
		}
	}
	key, err := signing.Generate(dataDir)
	if err != nil {
		return "", err
	}
	s.Keys.Holder.Set(key)
	if err := (settings.Store{DB: s.st().Server()}).Set(ctx, "signing_kid", key.KID()); err != nil {
		return "", err
	}
	return key.KID(), s.Spaces.Audit(ctx, actor, "rotate_key", nil, nil, key.KID())
}
