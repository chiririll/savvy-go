//go:build integration

package httpserver

import (
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"savvy-go/internal/auth/totp"
)

func TestTwoFactorEnableConfirmLoginRecovery(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("otp@test.com", "secret1", roleEditor)
	iss := a.issue(u, false)

	en := a.do("POST", "/api/auth/2fa/enable", map[string]any{"current_password": "secret1"}, iss.Token, iss.CSRF)
	if en.StatusCode != 200 {
		t.Fatalf("enable %d %v", en.StatusCode, decodeJSON(t, en))
	}
	enBody := decodeJSON(t, en)
	secret, _ := enBody["secret"].(string)
	if secret == "" || enBody["qr_code_url"] == nil {
		t.Fatalf("enable body %v", enBody)
	}

	bad := a.do("POST", "/api/auth/2fa/confirm", map[string]string{"code": "000000"}, iss.Token, iss.CSRF)
	if bad.StatusCode != 422 {
		t.Fatalf("bad confirm %d", bad.StatusCode)
	}
	bad.Body.Close()

	// An authenticator code works once, so each step below takes a later one.
	code := totp.At(secret, time.Now().Add(-30*time.Second))
	ok := a.do("POST", "/api/auth/2fa/confirm", map[string]string{"code": code}, iss.Token, iss.CSRF)
	if ok.StatusCode != 200 {
		t.Fatalf("confirm %d %v", ok.StatusCode, decodeJSON(t, ok))
	}
	confirmed := decodeJSON(t, ok)
	codes, _ := confirmed["recovery_codes"].([]any)
	if len(codes) != 8 {
		t.Fatalf("codes %v", confirmed)
	}

	login := a.do("POST", "/api/auth/login", map[string]string{"email": "otp@test.com", "password": "secret1"}, "", "")
	loginBody := decodeJSON(t, login)
	tok, _ := loginBody["two_factor_token"].(string)
	if tok == "" {
		t.Fatalf("login 2fa %v", loginBody)
	}

	verify := a.do("POST", "/api/auth/2fa/verify", map[string]any{
		"two_factor_token": tok, "code": totp.Now(secret),
	}, "", "")
	if verify.StatusCode != 200 {
		t.Fatalf("verify %d %v", verify.StatusCode, decodeJSON(t, verify))
	}
	if cookieNamed(verify, "svy_session") == nil {
		t.Fatal("expected session after 2fa")
	}
	verify.Body.Close()

	login2 := a.do("POST", "/api/auth/login", map[string]string{"email": "otp@test.com", "password": "secret1"}, "", "")
	tok2, _ := decodeJSON(t, login2)["two_factor_token"].(string)
	rec := a.do("POST", "/api/auth/2fa/verify", map[string]any{
		"two_factor_token": tok2, "code": codes[0],
	}, "", "")
	if rec.StatusCode != 200 {
		t.Fatalf("recovery verify %d %v", rec.StatusCode, decodeJSON(t, rec))
	}
	rec.Body.Close()

	st := a.do("GET", "/api/auth/2fa/status", nil, iss.Token, "")
	stBody := decodeJSON(t, st)
	if stBody["enabled"] != true {
		t.Fatalf("status %v", stBody)
	}
	if rem, _ := stBody["recovery_codes_remaining"].(float64); rem != 7 {
		t.Fatalf("remaining %v", stBody)
	}

	off := a.do(
		"POST",
		"/api/auth/2fa/disable",
		map[string]string{"code": totp.At(secret, time.Now().Add(30*time.Second))},
		iss.Token,
		iss.CSRF,
	)
	if off.StatusCode != 200 {
		t.Fatalf("disable %d %v", off.StatusCode, decodeJSON(t, off))
	}
}

// twoFactorUser is a user with 2FA on, and a way to get a fresh challenge.
func twoFactorUser(t *testing.T, a *testApp) (secret string, challenge func() string) {
	t.Helper()
	u := a.createUser("otp@test.com", "secret1", roleEditor)
	secret = "JBSWY3DPEHPK3PXP"
	if _,
		err := a.db.Exec(`UPDATE users SET two_factor_enabled = 1, two_factor_confirmed = 1, two_factor_secret = ? WHERE id `+
		`= ?`, secret, u.ID); err != nil {
		t.Fatal(err)
	}
	return secret, func() string {
		res := a.do("POST", "/api/auth/login", map[string]string{"email": "otp@test.com", "password": "secret1"}, "", "")
		tok, _ := decodeJSON(t, res)["two_factor_token"].(string)
		if tok == "" {
			t.Fatal("no challenge issued")
		}
		return tok
	}
}

func verify2FA(a *testApp, tok, code string) *http.Response {
	return a.do("POST", "/api/auth/2fa/verify", map[string]any{"two_factor_token": tok, "code": code}, "", "")
}

func TestTwoFactorChallengeAllowsFiveGuesses(t *testing.T) {
	a := newTestApp(t)
	secret, challenge := twoFactorUser(t, a)
	tok := challenge()

	for i := 1; i <= 4; i++ {
		res := verify2FA(a, tok, "000000")
		if res.StatusCode != 422 {
			t.Fatalf("guess %d: %d", i, res.StatusCode)
		}
		res.Body.Close()
	}
	// The fifth wrong code uses the challenge up: the SPA goes back to login.
	res := verify2FA(a, tok, "000000")
	if res.StatusCode != 401 {
		t.Fatalf("fifth guess: %d", res.StatusCode)
	}
	res.Body.Close()

	res = verify2FA(a, tok, totp.Now(secret))
	if res.StatusCode != 401 || cookieNamed(res, "svy_session") != nil {
		t.Fatalf("right code on a dead challenge: %d", res.StatusCode)
	}
	res.Body.Close()
}

func TestTwoFactorRightCodeFourthTry(t *testing.T) {
	a := newTestApp(t)
	secret, challenge := twoFactorUser(t, a)
	tok := challenge()
	for range 3 {
		verify2FA(a, tok, "000000").Body.Close()
	}
	res := verify2FA(a, tok, totp.Now(secret))
	if res.StatusCode != 200 || cookieNamed(res, "svy_session") == nil {
		t.Fatalf("fourth try: %d", res.StatusCode)
	}
	res.Body.Close()
}

func TestTwoFactorFailuresMakeAccountWait(t *testing.T) {
	// Many sign-ins here: only the 2FA lockout is under test, so the sign-in limits are off.
	a := newTestApp(t, WithConfig(testFile("[security]\nip_rate_limit = 0\naccount_rate_limit = 0\n")))
	secret, challenge := twoFactorUser(t, a)
	for range 2 {
		tok := challenge()
		for range 5 {
			verify2FA(a, tok, "000000").Body.Close()
		}
	}
	res := a.do("POST", "/api/auth/login", map[string]string{"email": "otp@test.com", "password": "secret1"}, "", "")
	body := decodeJSON(t, res)
	if res.StatusCode != 429 || body["two_factor_token"] != nil {
		t.Fatalf("login while locked: %d %v", res.StatusCode, body)
	}

	// After the delay a right code signs in, and that clears the count.
	if _,
		err := a.db.Exec(`UPDATE two_factor_failures SET locked_until = ?`,
		time.Now().Add(-time.Second).UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	ok := verify2FA(a, challenge(), totp.Now(secret))
	if ok.StatusCode != 200 {
		t.Fatalf("after the delay: %d", ok.StatusCode)
	}
	ok.Body.Close()
	var failures int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM two_factor_failures`).Scan(&failures)
	if failures != 0 {
		t.Fatalf("failures after success: %d", failures)
	}
}

// Two requests that answer one challenge at the same moment get one session.
func TestTwoFactorChallengeOneSessionRace(t *testing.T) {
	a := newTestApp(t)
	secret, challenge := twoFactorUser(t, a)
	tok := challenge()
	code := totp.Now(secret)

	var ok atomic.Int32
	var wg sync.WaitGroup
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res := verify2FA(a, tok, code)
			defer res.Body.Close()
			if res.StatusCode == 200 {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 1 {
		t.Fatalf("%d sessions for one challenge", ok.Load())
	}
}

// A code seen once cannot sign in a second time, even on a fresh challenge.
func TestTwoFactorRejectsAReplayedCode(t *testing.T) {
	a := newTestApp(t)
	secret, challenge := twoFactorUser(t, a)
	code := totp.Now(secret)

	res := verify2FA(a, challenge(), code)
	if res.StatusCode != 200 {
		t.Fatalf("first use: %d", res.StatusCode)
	}
	res.Body.Close()

	res = verify2FA(a, challenge(), code)
	if res.StatusCode != 422 || cookieNamed(res, "svy_session") != nil {
		t.Fatalf("replayed code: %d", res.StatusCode)
	}
	res.Body.Close()
}
