package spaces

import "testing"

func TestRoles(t *testing.T) {
	for _, c := range []struct {
		role         string
		valid, write bool
	}{
		{RoleAdmin, true, true},
		{RoleEditor, true, true},
		{RoleViewer, true, false},
		{"guest", false, false}, // a guest is a server role, not a space role
		{"", false, false},
	} {
		if got := ValidRole(c.role); got != c.valid {
			t.Errorf("ValidRole(%q) = %v", c.role, got)
		}
		if got := CanWrite(c.role); got != c.write {
			t.Errorf("CanWrite(%q) = %v", c.role, got)
		}
	}
}
