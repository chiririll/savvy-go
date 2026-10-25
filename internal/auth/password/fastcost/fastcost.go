// Package fastcost drops the bcrypt work factor to its minimum. Import it for effect from
// a _test.go file: tests hash many passwords, and the real cost makes them slow under -race.
package fastcost

import (
	"golang.org/x/crypto/bcrypt"

	"savvy-go/internal/auth/password"
)

func init() { password.SetCost(bcrypt.MinCost) }
