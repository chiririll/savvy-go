package auth

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"savvy-go/internal/auth/permissions"
)

func TestAPITokenIssueAndResolve(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	u := newUser(t, d, "a@test.com", RoleUser)
	tokens := APITokens{DB: d}
	sp := newSpaces(t, d, 3)

	raw, tok, err := tokens.Issue(ctx, u, "ci", Grant{Permissions: []string{"transactions:write", "transactions:read", "reports:read"}, SpaceIDs: []int64{sp[2], sp[0], sp[2]}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(raw, "svy_") || !strings.HasPrefix(raw, tok.Prefix) || len(tok.Prefix) != 8 {
		t.Errorf("token %q prefix %q", raw, tok.Prefix)
	}
	if tok.Kind != permissions.KindPersonal || tok.AllSpaces || !slices.Equal(tok.SpaceIDs, []int64{sp[0], sp[2]}) || !slices.Equal(tok.Permissions, []string{"reports:read", "transactions:write"}) {
		t.Errorf("grant was not normalised: %+v", tok)
	}

	got, err := tokens.Resolve(ctx, raw)
	if err != nil || got == nil || got.ID != tok.ID || got.User == nil || got.User.ID != u.ID {
		t.Fatalf("resolve: %+v, %v", got, err)
	}
	for _, bad := range []string{"", "svy_nope", strings.TrimPrefix(raw, "svy_"), raw + "x"} {
		if r, _ := tokens.Resolve(ctx, bad); r != nil {
			t.Errorf("%q resolved", bad)
		}
	}
	if list, _ := tokens.List(ctx, u.ID); len(list) != 1 || list[0].LastUsedAt == nil {
		t.Errorf("list: %+v", list)
	}
	if other := newUser(t, d, "b@test.com", RoleUser); true {
		if list, _ := tokens.List(ctx, other.ID); len(list) != 0 {
			t.Error("tokens leak to another user")
		}
	}
}

func TestExpiredAPITokenDoesNotResolve(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	u := newUser(t, d, "a@test.com", RoleUser)
	raw, _, err := APITokens{DB: d}.Issue(ctx, u, "old", Grant{Permissions: []string{"reports:read"}, AllSpaces: true}, ptr(time.Now().Add(-time.Minute)))
	if err != nil {
		t.Fatal(err)
	}
	if r, _ := (APITokens{DB: d}).Resolve(ctx, raw); r != nil {
		t.Error("an expired token resolved")
	}
	now := time.Now()
	if !(&APIToken{ExpiresAt: &now}).Expired(now) || (&APIToken{}).Expired(now) || (&APIToken{ExpiresAt: ptr(now.Add(time.Second))}).Expired(now) {
		t.Error("Expired")
	}
}

func TestRevokeAllAPITokensOfOneUser(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	u := newUser(t, d, "a@test.com", RoleUser)
	other := newUser(t, d, "b@test.com", RoleUser)
	tokens := APITokens{DB: d}
	sp := newSpaces(t, d, 1)
	grant := Grant{Permissions: []string{"reports:read"}, SpaceIDs: sp}

	first, _, _ := tokens.Issue(ctx, u, "one", grant, nil)
	second, _, _ := tokens.Issue(ctx, u, "two", Grant{Permissions: []string{"reports:read"}, AllSpaces: true}, nil)
	kept, _, _ := tokens.Issue(ctx, other, "kept", grant, nil)

	if err := tokens.RevokeAll(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{first, second} {
		if r, _ := tokens.Resolve(ctx, raw); r != nil {
			t.Error("a revoked token still resolves")
		}
	}
	if list, _ := tokens.List(ctx, u.ID); len(list) != 0 {
		t.Errorf("tokens left: %+v", list)
	}
	if r, _ := tokens.Resolve(ctx, kept); r == nil {
		t.Error("RevokeAll reached another user")
	}
	if err := tokens.RevokeAll(ctx, u.ID); err != nil {
		t.Errorf("revoking nothing is not an error: %v", err)
	}
}

func TestAPITokenGrantsAreChecked(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	u := newUser(t, d, "a@test.com", RoleUser)
	tokens := APITokens{DB: d}
	tests := []struct {
		name  string
		grant Grant
	}{
		{"no space", Grant{Permissions: []string{"reports:read"}}},
		{"an unknown resource", Grant{Permissions: []string{"payroll:read"}, AllSpaces: true}},
		{"an unknown level", Grant{Permissions: []string{"reports:admin"}, AllSpaces: true}},
		{"a typo that would grant less", Grant{Permissions: []string{"reports"}, AllSpaces: true}},
		{"a connected app with nothing", Grant{Kind: permissions.KindMCP, AllSpaces: true}},
		{"a connected app beyond its tools", Grant{Kind: permissions.KindMCP, Permissions: []string{"recurring:read"}, AllSpaces: true}},
	}
	for _, tc := range tests {
		_, _, err := tokens.Issue(ctx, u, "x", tc.grant, nil)
		if !errors.Is(err, permissions.ErrBadGrant) {
			t.Errorf("%s: err = %v, want permissions.ErrBadGrant", tc.name, err)
		}
	}
	if list, _ := tokens.List(ctx, u.ID); len(list) != 0 {
		t.Errorf("refused grants left %d tokens", len(list))
	}
}

func TestAPITokenLimit(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	u := newUser(t, d, "a@test.com", RoleUser)
	tokens := APITokens{DB: d}
	grant := Grant{Permissions: []string{"reports:read"}, AllSpaces: true}
	for range maxAPITokens {
		if _, _, err := tokens.Issue(ctx, u, "t", grant, nil); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := tokens.Issue(ctx, u, "one too many", grant, nil); !errors.Is(err, ErrTokenLimit) {
		t.Errorf("err = %v, want ErrTokenLimit", err)
	}
}

func TestUpdateGrantKeepsTheTokenAndItsKind(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	u := newUser(t, d, "a@test.com", RoleUser)
	other := newUser(t, d, "b@test.com", RoleUser)
	tokens := APITokens{DB: d}
	sp := newSpaces(t, d, 1)
	raw, tok, err := tokens.Issue(ctx, u, "app", Grant{Kind: permissions.KindMCP, Permissions: []string{"reports:read"}, SpaceIDs: sp}, nil)
	if err != nil {
		t.Fatal(err)
	}

	upd, err := tokens.UpdateGrant(ctx, u.ID, tok.ID, Grant{Permissions: []string{"accounts:read", "reports:read"}, AllSpaces: true})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Kind != permissions.KindMCP || !upd.AllSpaces || len(upd.Permissions) != 2 {
		t.Errorf("updated %+v", upd)
	}
	if r, _ := tokens.Resolve(ctx, raw); r == nil || !r.Allows("accounts", permissions.Read) {
		t.Error("the same token carries the new grant")
	}
	if _, err := tokens.UpdateGrant(ctx, u.ID, tok.ID, Grant{Permissions: []string{"recurring:read"}, AllSpaces: true}); !errors.Is(err, permissions.ErrBadGrant) {
		t.Errorf("a connected app cannot be widened beyond its tools: %v", err)
	}
	if _, err := tokens.UpdateGrant(ctx, other.ID, tok.ID, Grant{Permissions: []string{"reports:read"}, AllSpaces: true}); !errors.Is(err, ErrNotFound) {
		t.Errorf("another user's token: %v", err)
	}
	if _, err := tokens.Get(ctx, other.ID, tok.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get of another user's token: %v", err)
	}
}

func TestRevokeAPIToken(t *testing.T) {
	d := newDB(t)
	ctx := context.Background()
	u := newUser(t, d, "a@test.com", RoleUser)
	other := newUser(t, d, "b@test.com", RoleUser)
	tokens := APITokens{DB: d}
	raw, tok, _ := tokens.Issue(ctx, u, "x", Grant{Permissions: []string{"reports:read"}, AllSpaces: true}, nil)

	if err := tokens.Revoke(ctx, other.ID, tok.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("revoking someone else's token: %v", err)
	}
	if r, _ := tokens.Resolve(ctx, raw); r == nil {
		t.Fatal("another user revoked the token")
	}
	if err := tokens.Revoke(ctx, u.ID, tok.ID); err != nil {
		t.Fatal(err)
	}
	if r, _ := tokens.Resolve(ctx, raw); r != nil {
		t.Error("a revoked token resolved")
	}
	if err := tokens.Revoke(ctx, u.ID, tok.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("revoking twice: %v", err)
	}
}

func TestAPITokenAllows(t *testing.T) {
	tok := &APIToken{Permissions: []string{"transactions:delete", "reports:read", "accounts:write"}, SpaceIDs: []int64{2}}
	tests := []struct {
		resource, level string
		want            bool
	}{
		{"reports", permissions.Read, true},
		{"reports", permissions.Write, false},
		{"accounts", permissions.Read, true}, // write covers read
		{"accounts", permissions.Write, true},
		{"accounts", permissions.Delete, false},
		{"transactions", permissions.Read, true},
		{"transactions", permissions.Delete, true},
		{"budgets", permissions.Read, false},
		{"reports", "", false},
		{"reports", "root", false},
	}
	for _, tc := range tests {
		if got := tok.Allows(tc.resource, tc.level); got != tc.want {
			t.Errorf("Allows(%s, %s) = %v, want %v", tc.resource, tc.level, got, tc.want)
		}
	}
	if !tok.AllowsSpace(2) || tok.AllowsSpace(3) || !(&APIToken{AllSpaces: true}).AllowsSpace(99) {
		t.Error("AllowsSpace")
	}
}
