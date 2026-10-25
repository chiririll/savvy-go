//go:build integration

package httpserver

import (
	"testing"
)

func TestWebauthnOptionsCredentialCRUD(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("pk@test.com", "secret1", roleEditor)
	iss := a.issue(u, false)

	opts := a.do(
		"POST",
		"/api/auth/webauthn/register/options",
		map[string]any{"current_password": "secret1"},
		iss.Token,
		iss.CSRF,
	)
	if opts.StatusCode != 200 {
		t.Fatalf("register options %d %v", opts.StatusCode, decodeJSON(t, opts))
	}
	body := decodeJSON(t, opts)
	if body["token"] == nil {
		t.Fatalf("options %v", body)
	}
	opt, _ := body["options"].(map[string]any)
	if opt["challenge"] == nil || opt["rp"] == nil || opt["publicKey"] != nil {
		t.Fatalf("creation options %v", body["options"])
	}

	bad := a.do("POST", "/api/auth/webauthn/register/verify", map[string]any{
		"token": "nope", "response": map[string]any{"id": "x"},
	}, iss.Token, iss.CSRF)
	if bad.StatusCode != 422 {
		t.Fatalf("bad verify %d", bad.StatusCode)
	}
	bad.Body.Close()

	loginOpts := a.do("POST", "/api/auth/webauthn/login/options", map[string]any{}, "", "")
	if loginOpts.StatusCode != 200 {
		t.Fatalf("login options %d %v", loginOpts.StatusCode, decodeJSON(t, loginOpts))
	}
	loginOpts.Body.Close()

	_, _ = a.db.Exec(`INSERT INTO webauthn_credentials (user_id, credential_id, name, record, created_at)
		VALUES (?, 'cred-1', 'Laptop', '{}', '2026-01-01T00:00:00Z')`, u.ID)
	list := a.do("GET", "/api/auth/webauthn/credentials", nil, iss.Token, "")
	listed := decodeJSON(t, list)
	creds, _ := listed["credentials"].([]any)
	if len(creds) != 1 {
		t.Fatalf("list %v", listed)
	}
	id := int64(creds[0].(map[string]any)["id"].(float64))
	ren := a.do(
		"PATCH",
		"/api/auth/webauthn/credentials/"+itoa(id),
		map[string]string{"name": "Phone"},
		iss.Token,
		iss.CSRF,
	)
	if ren.StatusCode != 200 {
		t.Fatalf("rename %d", ren.StatusCode)
	}
	ren.Body.Close()
	del := a.do(
		"DELETE",
		"/api/auth/webauthn/credentials/"+itoa(id),
		map[string]any{"current_password": "secret1"},
		iss.Token,
		iss.CSRF,
	)
	if del.StatusCode != 200 {
		t.Fatalf("delete %d", del.StatusCode)
	}
	del.Body.Close()
}
