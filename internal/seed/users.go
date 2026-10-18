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

// demoUsers are the seeded accounts; role is the server role.
var demoUsers = []struct {
	key, name, email, pass, role string
}{
	{userAlex, "Alex Morgan", "admin@savvy.app", "password", auth.RoleAdmin},
	{userJordan, "Jordan Lee", "editor@savvy.app", "password", auth.RoleUser},
	{userSam, "Sam Rivera", "guest@savvy.app", "password", auth.RoleGuest},
	{userDemo, "Demo User", "demo@demo.com", "demo", auth.RoleUser},
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
		out[u.key] = created
	}
	return out, nil
}
