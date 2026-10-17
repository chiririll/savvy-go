package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func RandomString(n int) string {
	return randomFrom("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", n)
}

// randomFrom is n characters of alphabet, each uniform. A byte is dropped when
// it falls in the partial last run of 256 % len(alphabet), so no character is
// likelier than another. It panics when the system has no randomness.
func randomFrom(alphabet string, n int) string {
	limit := 256 - 256%len(alphabet)
	out := make([]byte, 0, n)
	buf := make([]byte, n+n/4+8)
	for len(out) < n {
		if _, err := rand.Read(buf); err != nil {
			panic(err)
		}
		for _, c := range buf {
			if int(c) < limit && len(out) < n {
				out = append(out, alphabet[int(c)%len(alphabet)])
			}
		}
	}
	return string(out)
}
