package sso

import (
	"testing"
)

func TestExtractIdentityFollowsTheClaimMappings(t *testing.T) {
	p := IdentityProvider{Preset: "entra"} // email: first of email, preferred_username, upn
	id := extractIdentity(p, map[string]any{
		"sub": "s", "preferred_username": "ann@corp.test", "email": "", "xms_edov": true, "name": "Ann", "groups": "single",
	}, nil)
	if id.Subject != "s" || id.Email != "ann@corp.test" || !id.EmailVerified || len(id.Groups) != 1 ||
		id.Groups[0] != "single" {
		t.Errorf("identity %+v", id)
	}

	custom := IdentityProvider{
		Preset:        "custom_oidc",
		ClaimMappings: map[string]any{"subject": "uid", "email": "mail", "email_verified": "mail_ok"},
	}
	id = extractIdentity(custom, map[string]any{"uid": float64(42), "mail": "x@test.com", "mail_ok": "true"}, nil)
	if id.Subject != "42" || id.Email != "x@test.com" || !id.EmailVerified {
		t.Errorf("custom mapping %+v", id)
	}

	id = extractIdentity(
		p,
		map[string]any{"sub": "s", "email": "claimed@test.com"},
		map[string]any{"email": "", "email_verified": false},
	)
	if id.Email != "" || id.EmailVerified {
		t.Errorf("overrides win over claims: %+v", id)
	}
	if id := extractIdentity(IdentityProvider{Preset: "google"}, map[string]any{"sub": "s", "email": "a@b.c"}, nil); id.EmailVerified {
		t.Error("an email the provider does not vouch for is unverified")
	}
}

func TestClaimValueAndBooleans(t *testing.T) {
	claims := map[string]any{"a": "", "b": "x", "n": nil}
	if claimValue(claims, nil) != nil || claimValue(claims, "b") != "x" ||
		claimValue(claims, []any{"a", "n", "b"}) != "x" ||
		claimValue(claims, []any{"zzz"}) != nil ||
		claimValue(claims, 5) != nil {
		t.Error("claimValue")
	}
	for in, want := range map[any]bool{true: true, false: false, float64(1): true, float64(0): false, "1": true, "TRUE": true, "yes": false, nil: false, "": false} {
		if asBoolAny(in) != want {
			t.Errorf("asBoolAny(%v) = %v", in, !want)
		}
	}
}
