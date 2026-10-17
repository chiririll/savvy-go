package sso

import (
	"testing"

	"savvy-go/internal/auth"
)

func TestRoleRules(t *testing.T) {
	groups := []any{"staff", "Admins"}
	tests := []struct {
		name          string
		op            string
		actual, value any
		want          bool
	}{
		{"equals", "equals", "admin", "admin", true},
		{"equals is exact", "equals", "admin", "Admin", false},
		{"equals across number and string", "equals", float64(3), "3", true},
		{"equals on a missing claim", "equals", nil, "", true},
		{"contains an item", "contains", groups, "staff", true},
		{"contains is not a substring match on lists", "contains", groups, "sta", false},
		{"contains a substring of a string", "contains", "team-admins-eu", "admins", true},
		{"one_of a list", "one_of", "b", []any{"a", "b"}, true},
		{"one_of with a list claim", "one_of", groups, []any{"x", "Admins"}, true},
		{"one_of none", "one_of", groups, []any{"x", "y"}, false},
		{"one_of a single value", "one_of", "a", "a", true},
		{"an unknown operator never matches", "regex", "a", "a", false},
	}
	for _, tc := range tests {
		if got := ruleMatches(tc.op, tc.actual, tc.value); got != tc.want {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestMatchRoleRule(t *testing.T) {
	id := NormalizedIdentity{Groups: []string{"finance"}, Raw: map[string]any{"dept": "eng", "level": float64(7)}}
	tests := []struct {
		name string
		rule map[string]any
		want string
	}{
		{
			"a group",
			map[string]any{"claim": "groups", "operator": "contains", "value": "finance", "role": "admin"},
			auth.RoleAdmin,
		},
		{"a plain claim, equals by default", map[string]any{"claim": "dept", "value": "eng", "role": "user"}, auth.RoleUser},
		{"a numeric claim", map[string]any{"claim": "level", "value": "7", "role": "guest"}, auth.RoleGuest},
		{"a retired role name", map[string]any{"claim": "dept", "value": "eng", "role": "read-write"}, auth.RoleUser},
		{"an invalid role is ignored", map[string]any{"claim": "dept", "value": "eng", "role": "owner"}, ""},
		{"no claim name", map[string]any{"value": "eng", "role": "admin"}, ""},
		{"no role", map[string]any{"claim": "dept", "value": "eng"}, ""},
		{"no match", map[string]any{"claim": "dept", "value": "ops", "role": "admin"}, ""},
	}
	for _, tc := range tests {
		if got := matchRoleRule(tc.rule, id); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
	typed := IdentityProvider{RoleMapping: []map[string]any{{"claim": "dept", "value": "eng", "role": "admin"}}}
	if got := (Service{}).mappedRole(typed, id); got != auth.RoleAdmin {
		t.Errorf("a typed rule list: %q", got)
	}
	if got := (Service{}).mappedRole(IdentityProvider{}, id); got != "" {
		t.Errorf("no mapping: %q", got)
	}
}

func TestStringify(t *testing.T) {
	for _, tc := range []struct {
		in   any
		want string
	}{{nil, ""}, {"x", "x"}, {float64(3), "3"}, {1.5, "1.5"}, {true, "true"}, {[]any{"a"}, `["a"]`}} {
		if got := stringify(tc.in); got != tc.want {
			t.Errorf("stringify(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
