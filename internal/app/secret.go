package app

import (
	"crypto/rand"
	"encoding/hex"
)

// randomSecret is a fresh secret for signing upload part URLs. It lives as long
// as the process: those URLs expire within minutes, and an upload interrupted
// by a restart starts over.
func randomSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err) // the system's random source is broken: nothing safe to do
	}
	return hex.EncodeToString(b)
}
