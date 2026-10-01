package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

func NewTOTPSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return strings.TrimRight(base32.StdEncoding.EncodeToString(b), "="), nil
}

func TOTPProvisioningURI(email, secret string) string {
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

func VerifyTOTP(secret, code string) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		key, err = base32.StdEncoding.DecodeString(strings.ToUpper(secret))
		if err != nil {
			return false
		}
	}
	now := time.Now().Unix() / 30
	for _, w := range []int64{-1, 0, 1} {
		if totpCode(key, now+w) == code {
			return true
		}
	}
	return false
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

// TOTPNow is exported for tests.
func TOTPNow(secret string) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		key, _ = base32.StdEncoding.DecodeString(strings.ToUpper(secret))
	}
	return totpCode(key, time.Now().Unix()/30)
}
