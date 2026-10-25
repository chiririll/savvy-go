//go:build integration

package httpserver

import (
	"fmt"
	"testing"
	"time"
)

// freezeLimits stops the clock the attempt limiters read: a slow run must not refill a bucket
// and let a request through that the test expects to be refused.
func freezeLimits(a *testApp) {
	now := time.Now()
	a.s.guard.SetClock(func() time.Time { return now })
}

func tryLogin(a *testApp, email, pass string) int {
	res := a.do("POST", "/api/auth/login", map[string]string{"email": email, "password": pass}, "", "")
	res.Body.Close()
	return res.StatusCode
}

func TestLoginIsLimitedPerIP(t *testing.T) {
	a := newTestApp(t, WithConfig(testFile("[security]\nip_rate_limit = 10\naccount_rate_limit = 0\n")))
	for i := 0; i < 10; i++ {
		if got := tryLogin(a, fmt.Sprintf("u%d@test.com", i), "wrong-pass"); got != 422 {
			t.Fatalf("attempt %d: %d, want 422", i+1, got)
		}
	}
	res := a.do("POST", "/api/auth/login", map[string]string{"email": "x@test.com", "password": "wrong-pass"}, "", "")
	res.Body.Close()
	if res.StatusCode != 429 {
		t.Fatalf("11th attempt: %d, want 429", res.StatusCode)
	}
	if res.Header.Get("Retry-After") == "" {
		t.Error("429 without Retry-After")
	}
	// The same address is also refused on the other guessable routes.
	bs := a.do("POST", "/api/auth/bootstrap/verify", map[string]string{"bootstrap_password": "x"}, "", "")
	bs.Body.Close()
	if bs.StatusCode != 429 {
		t.Errorf("bootstrap verify shares the count: %d", bs.StatusCode)
	}
	// Reading who is signed in is not a guess and stays open.
	st := a.do("GET", "/api/auth/status", nil, "", "")
	st.Body.Close()
	if st.StatusCode != 200 {
		t.Errorf("status %d", st.StatusCode)
	}
}

func TestLoginIsLimitedPerAccount(t *testing.T) {
	a := newTestApp(t, WithConfig(testFile("[security]\nip_rate_limit = 0\naccount_rate_limit = 5\n")))
	a.createUser("victim@test.com", "right-pass1", roleEditor)
	for i := 0; i < 5; i++ {
		if got := tryLogin(a, "Victim@Test.com", "wrong-pass"); got != 422 {
			t.Fatalf("attempt %d: %d, want 422", i+1, got)
		}
	}
	// Even the right password waits: the count is of attempts, not failures.
	if got := tryLogin(a, " victim@test.com ", "right-pass1"); got != 429 {
		t.Fatalf("6th attempt (other spelling of the email): %d, want 429", got)
	}
	if got := tryLogin(a, "other@test.com", "wrong-pass"); got != 422 {
		t.Errorf("another account is not affected: %d", got)
	}
}

func TestRegisterNeedsEightCharacters(t *testing.T) {
	a := newTestApp(t)
	res := a.do("POST", "/api/auth/register", map[string]string{
		"name": "Admin", "email": "admin@test.com", "password": "1234567", "bootstrap_password": "x",
	}, "", "")
	body := decodeJSON(t, res)
	if res.StatusCode != 422 || body["errors"] == nil || fmt.Sprint(body["errors"]) == "" {
		t.Fatalf("7 characters: %d %v", res.StatusCode, body)
	}
	if errs, _ := body["errors"].(map[string]any); errs["password"] == nil {
		t.Errorf("no password error: %v", body)
	}
}

func TestInvitationLinksAreLimitedPerIP(t *testing.T) {
	a := newTestApp(t, WithConfig(testFile("[security]\nip_rate_limit = 3\naccount_rate_limit = 0\n")))
	for i := 0; i < 3; i++ {
		res := a.do("GET", "/api/invitations/not-a-token", nil, "", "")
		res.Body.Close()
		if res.StatusCode == 429 {
			t.Fatalf("attempt %d limited early", i+1)
		}
	}
	res := a.do("GET", "/api/invitations/another-guess", nil, "", "")
	res.Body.Close()
	if res.StatusCode != 429 {
		t.Fatalf("4th guess: %d, want 429", res.StatusCode)
	}
}
