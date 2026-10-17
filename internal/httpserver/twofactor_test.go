package httpserver

import (
	"testing"

	"savvy-go/internal/auth/totp"
)

func TestTwoFactorEnableConfirmLoginAndRecovery(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("otp@test.com", "secret1", roleEditor)
	iss := a.issue(u, false)

	en := a.do("POST", "/api/auth/2fa/enable", map[string]any{}, iss.Token, iss.CSRF)
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

	code := totp.Now(secret)
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

	off := a.do("POST", "/api/auth/2fa/disable", map[string]string{"code": totp.Now(secret)}, iss.Token, iss.CSRF)
	if off.StatusCode != 200 {
		t.Fatalf("disable %d %v", off.StatusCode, decodeJSON(t, off))
	}
}
