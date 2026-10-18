// Package password hashes and checks account passwords with bcrypt.
package password

import (
	"fmt"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// cost is the bcrypt work factor. Tests lower it with SetCost.
var cost = bcrypt.DefaultCost

// SetCost changes the bcrypt work factor for hashes made from now on. It is for test
// packages, which hash many passwords; call it before any test runs.
func SetCost(c int) { cost = c }

// MinLength is the fewest characters any password path accepts: sign-up, change, reset,
// invitation and the admin form.
const MinLength = 8

// TooShort reports whether plain is under MinLength.
func TooShort(plain string) bool { return len(plain) < MinLength }

// TooShortMessage is the validation message for a password that is.
var TooShortMessage = fmt.Sprintf("The password must be at least %d characters.", MinLength)

func Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	return string(b), err
}

func Check(hash, plain string) bool {
	if hash == "" || plain == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// dummyHash is a bcrypt hash at the same cost as real ones, made once on first
// use, that no input is meant to match.
var dummyHash = sync.OnceValue(func() string {
	b, err := bcrypt.GenerateFromPassword([]byte("no account has this password"), cost)
	if err != nil {
		panic(err)
	}
	return string(b)
})

// Dummy spends the time of one real check and reports false. Sign-in calls it
// when there is no password to check, so a missing account costs the same as a
// wrong password.
func Dummy() bool {
	_ = bcrypt.CompareHashAndPassword([]byte(dummyHash()), []byte("x"))
	return false
}

// CheckConstantTime is Check that always does one bcrypt compare: with no hash (nil
// or empty) or an empty plain it runs Dummy instead of returning early.
func CheckConstantTime(hash *string, plain string) bool {
	if hash == nil || *hash == "" || plain == "" {
		return Dummy()
	}
	return Check(*hash, plain)
}
