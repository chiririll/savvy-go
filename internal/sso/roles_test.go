package sso

import (
	"testing"

	"savvy-go/internal/auth"
)

// Role mappings written for the retired read-write / read-only roles
// give the server role user.
func TestMappedRoleRetiredRolesBecomeUser(t *testing.T) {
	id := NormalizedIdentity{Email: "a@b.c", Groups: []string{"finance"}}
	for _, old := range []string{"read-write", "read-only"} {
		p := IdentityProvider{
			RoleMapping: []any{map[string]any{"claim": "groups", "operator": "contains", "value": "finance", "role": old}},
		}
		if got := (Service{}).mappedRole(p, id); got != auth.RoleUser {
			t.Fatalf("%s maps to %q", old, got)
		}
	}
	p := IdentityProvider{
		RoleMapping: []any{map[string]any{"claim": "groups", "operator": "contains", "value": "finance", "role": "admin"}},
	}
	if got := (Service{}).mappedRole(p, id); got != auth.RoleAdmin {
		t.Fatalf("admin maps to %q", got)
	}
}
