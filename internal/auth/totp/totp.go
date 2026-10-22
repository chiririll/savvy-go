// Package totp makes and checks the one-time codes of authenticator apps (RFC 6238).
package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

func NewSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return strings.TrimRight(base32.StdEncoding.EncodeToString(b), "="), nil
}

func ProvisioningURI(email, secret string) string {
	u := url.URL{Scheme: "otpauth", Host: "totp", Path: "/Savvy:" + email}
	q := u.Query()
	q.Set("secret", secret)
	q.Set("issuer", "Savvy")
	q.Set("algorithm", "SHA1")
	q.Set("digits", "6")
	q.Set("period", "30")
	u.RawQuery = q.Encode()
	return u.String()
}

// VerifyStep reports whether code is right for secret at now, from one step
// before to one step after, and which 30 second step it matched. A caller that
// stores the step can refuse a code it has already accepted.
func VerifyStep(secret, code string, now time.Time) (step int64, ok bool) {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return 0, false
	}
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		key, err = base32.StdEncoding.DecodeString(strings.ToUpper(secret))
		if err != nil {
			return 0, false
		}
	}
	current := now.Unix() / 30
	// Every step is compared, so the time taken does not tell which one matched.
	for _, w := range []int64{-1, 0, 1} {
		if subtle.ConstantTimeCompare([]byte(totpCode(key, current+w)), []byte(code)) == 1 && !ok {
			step, ok = current+w, true
		}
	}
	return step, ok
}

func totpCode(key []byte, counter int64) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(counter))
	mac := hmac.New(sha1.New, key)
	mac.Write(buf[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	n := int32(sum[off]&0x7f)<<24 | int32(sum[off+1])<<16 | int32(sum[off+2])<<8 | int32(sum[off+3])
	return fmt.Sprintf("%06d", n%1000000)
}

// At is the code an authenticator shows at t. Exported for tests.
func At(secret string, t time.Time) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		key, _ = base32.StdEncoding.DecodeString(strings.ToUpper(secret))
	}
	return totpCode(key, t.Unix()/30)
}

// Now is the code shown right now. Exported for tests.
func Now(secret string) string { return At(secret, time.Now()) }
