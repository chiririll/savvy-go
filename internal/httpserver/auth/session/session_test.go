package session

import (
	"testing"

	"savvy-go/internal/auth"
	"savvy-go/internal/auth/password"
)

func TestCredentialsOKChecksOnceForEveryKindOfAccount(t *testing.T) {
	hash, _ := password.Hash("secret12")
	empty := ""
	cases := map[string]struct {
		user    *auth.User
		plain   string
		want    bool
		wantNil bool // the checker should get no hash
	}{
		"unknown email":   {nil, "secret12", false, true},
		"sso only":        {&auth.User{Password: &hash, IsSSOOnly: true}, "secret12", false, true},
		"inactive":        {&auth.User{Password: &empty}, "secret12", false, true},
		"no password":     {&auth.User{}, "secret12", false, true},
		"wrong password":  {&auth.User{Password: &hash}, "nope-nope", false, false},
		"right password":  {&auth.User{Password: &hash}, "secret12", true, false},
		"empty submitted": {&auth.User{Password: &hash}, "", false, false},
	}
	for name, c := range cases {
		calls := 0
		check := func(h *string, plain string) bool {
			calls++
			if (h == nil) != c.wantNil {
				t.Errorf("%s: hash given = %v, want nil = %v", name, h != nil, c.wantNil)
			}
			return password.CheckOrBurn(h, plain)
		}
		if got := credentialsOK(check, c.user, c.plain); got != c.want {
			t.Errorf("%s: got %v, want %v", name, got, c.want)
		}
		if calls != 1 {
			t.Errorf("%s: %d password checks, want exactly 1", name, calls)
		}
	}
}

func TestCredentialsOKRejectsEvenIfCheckerAccepts(t *testing.T) {
	yes := func(*string, string) bool { return true }
	if credentialsOK(yes, nil, "x") {
		t.Error("an unknown account signed in")
	}
}
