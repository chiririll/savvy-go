package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type laravelPayload struct {
	IV    string `json:"iv"`
	Value string `json:"value"`
	MAC   string `json:"mac"`
	Tag   string `json:"tag"`
}

// LooksLikeLaravelEncrypted reports Laravel's encrypt() envelope
// (base64 of {"iv","value","mac"}).
func LooksLikeLaravelEncrypted(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || !strings.ContainsAny(s, "+/=") {
		return false
	}
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return false
	}
	var p laravelPayload
	if json.Unmarshal(raw, &p) != nil {
		return false
	}
	return p.IV != "" && p.Value != "" && p.MAC != ""
}

// ParseLaravelKey decodes APP_KEY (`base64:…` or raw bytes).
func ParseLaravelKey(appKey string) ([]byte, error) {
	appKey = strings.TrimSpace(appKey)
	if appKey == "" {
		return nil, fmt.Errorf("empty APP_KEY")
	}
	if strings.HasPrefix(appKey, "base64:") {
		return base64.StdEncoding.DecodeString(strings.TrimPrefix(appKey, "base64:"))
	}
	return []byte(appKey), nil
}

// DecryptLaravel decrypts a Laravel Encrypter payload (AES-256-CBC or AES-256-GCM)
// and unwraps a PHP-serialized string when present.
func DecryptLaravel(appKey, ciphertext string) (string, error) {
	key, err := ParseLaravelKey(appKey)
	if err != nil {
		return "", err
	}
	if len(key) != 16 && len(key) != 32 {
		return "", fmt.Errorf("APP_KEY must be 16 or 32 bytes")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(ciphertext))
	if err != nil {
		return "", err
	}
	var p laravelPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(p.IV + p.Value))
	expect := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expect), []byte(strings.ToLower(p.MAC))) && !hmac.Equal([]byte(expect), []byte(p.MAC)) {
		return "", fmt.Errorf("invalid mac")
	}
	iv, err := base64.StdEncoding.DecodeString(p.IV)
	if err != nil {
		return "", err
	}
	val, err := base64.StdEncoding.DecodeString(p.Value)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	var plain []byte
	if strings.TrimSpace(p.Tag) != "" {
		tag, err := base64.StdEncoding.DecodeString(p.Tag)
		if err != nil {
			return "", err
		}
		gcm, err := cipher.NewGCMWithNonceSize(block, len(iv))
		if err != nil {
			return "", err
		}
		plain, err = gcm.Open(nil, iv, append(val, tag...), nil)
		if err != nil {
			return "", err
		}
	} else {
		if len(val)%aes.BlockSize != 0 || len(iv) != aes.BlockSize {
			return "", fmt.Errorf("invalid ciphertext")
		}
		mode := cipher.NewCBCDecrypter(block, iv)
		plain = make([]byte, len(val))
		mode.CryptBlocks(plain, val)
		plain, err = pkcs7unpad(plain)
		if err != nil {
			return "", err
		}
	}
	return phpUnserializeString(string(plain)), nil
}

// EncryptLaravel produces a Laravel-compatible AES-256-CBC payload (serialize=true).
func EncryptLaravel(appKey, plaintext string) (string, error) {
	key, err := ParseLaravelKey(appKey)
	if err != nil {
		return "", err
	}
	if len(key) != 16 && len(key) != 32 {
		return "", fmt.Errorf("APP_KEY must be 16 or 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}
	raw := pkcs7pad([]byte(phpSerializeString(plaintext)), aes.BlockSize)
	mode := cipher.NewCBCEncrypter(block, iv)
	dst := make([]byte, len(raw))
	mode.CryptBlocks(dst, raw)
	ivB64 := base64.StdEncoding.EncodeToString(iv)
	valB64 := base64.StdEncoding.EncodeToString(dst)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(ivB64 + valB64))
	payload, err := json.Marshal(laravelPayload{
		IV: ivB64, Value: valB64, MAC: hex.EncodeToString(mac.Sum(nil)), Tag: "",
	})
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(payload), nil
}

func phpSerializeString(s string) string {
	return "s:" + strconv.Itoa(len(s)) + ":\"" + s + "\";"
}

func phpUnserializeString(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "s:") {
		return s
	}
	rest := s[2:]
	colon := strings.IndexByte(rest, ':')
	if colon < 0 {
		return s
	}
	n, err := strconv.Atoi(rest[:colon])
	if err != nil || n < 0 {
		return s
	}
	body := rest[colon+1:]
	if len(body) < n+3 || body[0] != '"' {
		return s
	}
	return body[1 : 1+n]
}

func pkcs7pad(b []byte, block int) []byte {
	n := block - (len(b) % block)
	out := make([]byte, len(b)+n)
	copy(out, b)
	for i := len(b); i < len(out); i++ {
		out[i] = byte(n)
	}
	return out
}

func pkcs7unpad(b []byte) ([]byte, error) {
	if len(b) == 0 {
		return nil, fmt.Errorf("empty")
	}
	n := int(b[len(b)-1])
	if n == 0 || n > len(b) {
		return nil, fmt.Errorf("bad padding")
	}
	for i := 0; i < n; i++ {
		if b[len(b)-1-i] != byte(n) {
			return nil, fmt.Errorf("bad padding")
		}
	}
	return b[:len(b)-n], nil
}
