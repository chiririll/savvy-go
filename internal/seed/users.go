package seed

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"savvy-go/internal/auth"
)

// Demo user keys, used to wire up spaces and memberships.
const (
	userAlex   = "alex"
	userJordan = "jordan"
	userSam    = "sam"
)

// demoUser is a seeded account; role is the server role. An unverified user has
// not proven the address, which shows once mail is configured.
type demoUser struct {
	key, name, email, pass, role string
	unverified                   bool
}

var demoUsers = []demoUser{
	{userAlex, "Alex Morgan", "admin@savvy.app", "password", auth.RoleAdmin, false},
	{userJordan, "Jordan Lee", "editor@savvy.app", "password", auth.RoleUser, false},
	{userSam, "Sam Rivera", "guest@savvy.app", "password", auth.RoleGuest, true},
}

// copyOf returns u as the n-th copy: "admin@savvy.app" becomes "admin1@savvy.app",
// the key "alex" becomes "alex1". Copy 0 is u itself.
func (u demoUser) copyOf(n int) demoUser {
	if n == 0 {
		return u
	}
	sfx := strconv.Itoa(n)
	local, domain, _ := strings.Cut(u.email, "@")
	u.key += sfx
	u.name += " " + sfx
	u.email = local + sfx + "@" + domain
	return u
}

// createDemoUsers creates copy n of the users (0 is the regular set) and
// returns the created accounts by the base key, with the users of the copy.
func createDemoUsers(ctx context.Context, users auth.Users, n int) (map[string]*auth.User, []demoUser, error) {
	out := map[string]*auth.User{}
	var made []demoUser
	for _, base := range demoUsers {
		u := base.copyOf(n)
		pass := u.pass
		created, err := users.Create(ctx, u.name, u.email, &pass, u.role)
		if err != nil {
			return nil, nil, fmt.Errorf("user %s: %w", u.email, err)
		}
		if !u.unverified {
			if err := users.SetEmailVerified(ctx, created.ID, true); err != nil {
				return nil, nil, fmt.Errorf("verify %s: %w", u.email, err)
			}
		}
		out[base.key] = created
		made = append(made, u)
	}
	return out, made, nil
}
