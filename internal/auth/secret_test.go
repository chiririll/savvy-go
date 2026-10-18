package auth

import (
	"strings"
	"testing"
)

func TestRandomStringTokenHash(t *testing.T) {
	a, b := RandomString(32), RandomString(32)
	if len(a) != 32 || a == b || strings.Trim(a, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789") != "" {
		t.Errorf("random strings %q %q", a, b)
	}
	if HashToken("x") != HashToken("x") || HashToken("x") == HashToken("y") || len(HashToken("x")) != 64 {
		t.Error("token hashes are stable, distinct and hex sha-256")
	}
}
