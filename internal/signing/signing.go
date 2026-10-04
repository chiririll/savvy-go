// Package signing holds the server's Ed25519 key. The server signs what may
// leave it and come back (backup manifests, cross-space transfers), so it can
// later tell its own output from a modified or foreign one. The private key
// lives in a file in the data directory, never in a database, so a space
// backup cannot carry it.
package signing

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	keyDir  = "keys"
	keyFile = "server.ed25519"
)

// ErrKeyMissing is returned when the key file is gone although the server
// has signed with it before: generating a new one silently would make every
// earlier signature unverifiable.
var ErrKeyMissing = errors.New("the server signing key is missing; restore " +
	filepath.Join(keyDir, keyFile) + " from the data volume or rotate the key explicitly")

// Key is the server's signing key.
type Key struct {
	priv ed25519.PrivateKey
}

// Path is where the key file lives under the data directory.
func Path(dataDir string) string { return filepath.Join(dataDir, keyDir, keyFile) }

// Load reads the key from dataDir. When the file does not exist it creates
// one, unless used reports that a key was in use before (ErrKeyMissing).
func Load(dataDir string, used bool) (*Key, error) {
	raw, err := os.ReadFile(Path(dataDir))
	switch {
	case err == nil:
		return Parse(raw)
	case !os.IsNotExist(err):
		return nil, err
	case used:
		return nil, ErrKeyMissing
	}
	return Generate(dataDir)
}

// Generate creates a new key and writes it to dataDir, replacing any old one.
func Generate(dataDir string) (*Key, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	if err := Write(dataDir, priv.Seed()); err != nil {
		return nil, err
	}
	return &Key{priv: priv}, nil
}

// Write stores a key seed (from Seed or a server backup) in dataDir.
func Write(dataDir string, seed []byte) error {
	if len(seed) != ed25519.SeedSize {
		return fmt.Errorf("signing key: want %d bytes, got %d", ed25519.SeedSize, len(seed))
	}
	path := Path(dataDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(seed)), 0o600)
}

// Parse reads a key file's contents.
func Parse(raw []byte) (*Key, error) {
	seed, err := base64.StdEncoding.DecodeString(string(raw))
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, errors.New("signing key file is malformed")
	}
	return &Key{priv: ed25519.NewKeyFromSeed(seed)}, nil
}

// Seed is the private key material, for the server backup only.
func (k *Key) Seed() []byte { return k.priv.Seed() }

// Public is the public key.
func (k *Key) Public() ed25519.PublicKey { return k.priv.Public().(ed25519.PublicKey) }

// KID identifies a public key: the first 16 hex digits of its SHA-256.
func KID(pub ed25519.PublicKey) string {
	sum := sha256.Sum256(pub)
	return hex.EncodeToString(sum[:8])
}

// KID identifies this key.
func (k *Key) KID() string { return KID(k.Public()) }

// Sign signs msg.
func (k *Key) Sign(msg []byte) []byte { return ed25519.Sign(k.priv, msg) }

// Verify reports whether sig is a signature of msg by pub.
func Verify(pub ed25519.PublicKey, msg, sig []byte) bool {
	return len(pub) == ed25519.PublicKeySize && ed25519.Verify(pub, msg, sig)
}

// Holder keeps the current key; a server restore replaces it.
type Holder struct {
	mu  sync.RWMutex
	key *Key
}

// NewHolder holds k.
func NewHolder(k *Key) *Holder { return &Holder{key: k} }

// Key is the current key.
func (h *Holder) Key() *Key {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.key
}

// Set replaces the current key.
func (h *Holder) Set(k *Key) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.key = k
}
