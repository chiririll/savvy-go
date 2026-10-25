package seed

import (
	"context"
	"fmt"

	"savvy-go/internal/auth"
)

// Demo user keys, used to wire up spaces and memberships.
const (
	userAlex   = "alex"
	userJordan = "jordan"
	userSam    = "sam"
	userDemo   = "demo"
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
	{userDemo, "Demo User", "demo@demo.com", "demo", auth.RoleUser, false},
}

// createDemoUsers creates the users and returns them by key.
func createDemoUsers(ctx context.Context, users auth.Users) (map[string]*auth.User, error) {
	out := map[string]*auth.User{}
	for _, u := range demoUsers {
		pass := u.pass
		created, err := users.Create(ctx, u.name, u.email, &pass, u.role)
		if err != nil {
			return nil, fmt.Errorf("user %s: %w", u.email, err)
		}
		if !u.unverified {
			if err := users.SetEmailVerified(ctx, created.ID, true); err != nil {
				return nil, fmt.Errorf("verify %s: %w", u.email, err)
			}
		}
		out[u.key] = created
	}
	return out, nil
}
