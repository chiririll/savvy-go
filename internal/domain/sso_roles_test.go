package domain

import (
	"testing"

	"savvy-go/internal/auth"
)

// P13: role mappings written for the retired read-write / read-only roles
// give the server role user.
func TestP13RetiredSSORolesMapToUser(t *testing.T) {
	id := NormalizedIdentity{Email: "a@b.c", Groups: []string{"finance"}}
	for _, old := range []string{"read-write", "read-only"} {
		p := IdentityProvider{RoleMapping: []any{map[string]any{"claim": "groups", "operator": "contains", "value": "finance", "role": old}}}
		if got := (SSO{}).mappedRole(p, id); got != auth.RoleUser {
			t.Fatalf("%s maps to %q", old, got)
		}
	}
	p := IdentityProvider{RoleMapping: []any{map[string]any{"claim": "groups", "operator": "contains", "value": "finance", "role": "admin"}}}
	if got := (SSO{}).mappedRole(p, id); got != auth.RoleAdmin {
		t.Fatalf("admin maps to %q", got)
	}
}
