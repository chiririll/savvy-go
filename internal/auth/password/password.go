// Package password hashes and checks account passwords with bcrypt.
package password

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 10

// MinLength is the fewest characters any password path accepts: sign-up, change, reset,
// invitation and the admin form.
const MinLength = 8

// TooShort reports whether plain is under MinLength.
func TooShort(plain string) bool { return len(plain) < MinLength }

// TooShortMessage is the validation message for a password that is.
var TooShortMessage = fmt.Sprintf("The password must be at least %d characters.", MinLength)

func Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	return string(b), err
}

func Check(hash, plain string) bool {
	if hash == "" || plain == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
