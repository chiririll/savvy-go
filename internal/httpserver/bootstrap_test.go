package httpserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"savvy-go/internal/currencydata"
	"savvy-go/internal/firstrun"
)

func (a *testApp) register(password string, extra map[string]string) *http.Response {
	body := map[string]string{"name": "A", "email": "a@a.com", "password": "secret12"}
	if password != "" {
		body["bootstrap_password"] = password
	}
	for k, v := range extra {
		body[k] = v
	}
	return a.do("POST", "/api/auth/register", body, "", "")
}

func fieldErrors(t *testing.T, res *http.Response, field string) {
	t.Helper()
	body := decodeJSON(t, res)
	errs, _ := body["errors"].(map[string]any)
	if res.StatusCode != http.StatusUnprocessableEntity || errs[field] == nil {
		t.Fatalf("want 422 with an error on %q, got %d %v", field, res.StatusCode, body)
	}
}

func TestRegisterNeedsTheBootstrapPassword(t *testing.T) {
	a := newTestApp(t)

	fieldErrors(t, a.register("", nil), "bootstrap_password")
	fieldErrors(t, a.register("AAAAA-AAAAA-AAAAA-AAAAA-AAAAA-AAAAA", nil), "bootstrap_password")

	var n int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	if n != 0 {
		t.Fatalf("%d users created without the password", n)
	}
	res := a.register(a.bootstrap.Password(), nil)
	status(t, res, http.StatusCreated, "register with the password")
}

// appWithGuard is a fresh server with the given bootstrap guard, or none when
// g is nil. It has no shared test space.
func appWithGuard(t *testing.T, g *firstrun.Guard) *testApp {
	t.Helper()
	cfg, _ := testConfig(t)
	st := openTestStore(t, cfg)
	opts := []Option{WithRates(currencydata.NewOffline())}
	if g != nil {
		opts = append(opts, WithBootstrap(g))
	}
	s := newServer(cfg, st, testKeys(t, cfg), opts...)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return &testApp{t: t, db: st.SQL(), store: st, srv: ts, s: s, svc: servicesOf(s), client: ts.Client(), bootstrap: g}
}

func TestRegisterWithoutAGuardIsClosed(t *testing.T) {
	a := appWithGuard(t, nil)
	status(t, a.register("anything", nil), http.StatusForbidden, "register with no guard")
}

func TestRegisterIsClosedOnceUserExists(t *testing.T) {
	a := newTestApp(t)
	pass := a.bootstrap.Password()
	status(t, a.register(pass, nil), http.StatusCreated, "first")
	status(t, a.register(pass, map[string]string{"email": "b@b.com"}), http.StatusForbidden, "second")
	if a.bootstrap.Active() {
		t.Fatal("password still valid after the first user")
	}
	if _, err := os.Stat(filepath.Join(a.svc.Cfg.DataDir, firstrun.FileName)); !os.IsNotExist(err) {
		t.Fatalf("password file still there: %v", err)
	}
}

func TestRegisterClosedOnceUsersExist(t *testing.T) {
	a := newTestApp(t)
	a.createUser("someone@test.com", "secret1", "user")
	status(t, a.register(a.bootstrap.Password(), nil), http.StatusForbidden, "users present")
}

func TestInvalidFieldsDoNotSpendPassword(t *testing.T) {
	a := newTestApp(t)
	res := a.register(a.bootstrap.Password(), map[string]string{"email": "not-an-email"})
	fieldErrors(t, res, "email")
	if !a.bootstrap.Active() {
		t.Fatal("a validation error consumed the password")
	}
	status(t, a.register(a.bootstrap.Password(), nil), http.StatusCreated, "retry")
}

func TestParallelRegistrationsCreateOneAdmin(t *testing.T) {
	a := newTestApp(t)
	pass := a.bootstrap.Password()
	var wg sync.WaitGroup
	codes := make(chan int, 10)
	for i := range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res := a.register(pass, map[string]string{"email": string(rune('a'+i)) + "@x.com"})
			res.Body.Close()
			codes <- res.StatusCode
		}()
	}
	wg.Wait()
	close(codes)
	created := 0
	for c := range codes {
		if c == http.StatusCreated {
			created++
		}
	}
	var n int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	if created != 1 || n != 1 {
		t.Fatalf("created %d responses, %d users", created, n)
	}
}

func TestStatusCarriesTheBootstrapHint(t *testing.T) {
	a := newTestApp(t)
	body := decodeJSON(t, a.do("GET", "/api/auth/status", nil, "", ""))
	hint, _ := body["bootstrap_hint"].(map[string]any)
	if body["needs_registration"] != true || hint["environment"] != "binary" ||
		hint["file"] != filepath.Join(a.svc.Cfg.DataDir, firstrun.FileName) {
		t.Fatalf("status %v", body)
	}
	if _, leaked := hint["password"]; leaked || len(hint) != 2 {
		t.Fatalf("hint must carry only environment and file: %v", hint)
	}

	status(t, a.register(a.bootstrap.Password(), nil), http.StatusCreated, "register")
	body = decodeJSON(t, a.do("GET", "/api/auth/status", nil, "", ""))
	if body["needs_registration"] != false || body["bootstrap_hint"] != nil {
		t.Fatalf("status after the first user %v", body)
	}
}

func TestStatusHintFollowsTheEnvironment(t *testing.T) {
	for _, env := range []firstrun.Env{firstrun.Docker, firstrun.Systemd} {
		cfg, _ := testConfig(t)
		a := appWithGuard(t, firstrun.New(cfg.DataDir, env))
		hint, _ := decodeJSON(t, a.do("GET", "/api/auth/status", nil, "", ""))["bootstrap_hint"].(map[string]any)
		if hint["environment"] != string(env) {
			t.Errorf("%s: hint %v", env, hint)
		}
	}
}

func (a *testApp) verifyBootstrap(password string) *http.Response {
	return a.do("POST", "/api/auth/bootstrap/verify", map[string]string{"bootstrap_password": password}, "", "")
}

func TestVerifyBootstrapPassword(t *testing.T) {
	a := newTestApp(t)
	fieldErrors(t, a.verifyBootstrap(""), "bootstrap_password")
	fieldErrors(t, a.verifyBootstrap("AAAAA-AAAAA-AAAAA-AAAAA-AAAAA-AAAAA"), "bootstrap_password")
	status(t, a.verifyBootstrap(a.bootstrap.Password()), http.StatusNoContent, "right password")

	var n int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	if n != 0 || !a.bootstrap.Active() {
		t.Fatalf("verify must create nothing and keep the password (users=%d)", n)
	}
	status(t, a.register(a.bootstrap.Password(), nil), http.StatusCreated, "register after verify")
}

func TestVerifyBootstrapClosedWhenUsersExist(t *testing.T) {
	a := newTestApp(t)
	pass := a.bootstrap.Password()
	a.createUser("someone@test.com", "secret1", "user")
	status(t, a.verifyBootstrap(pass), http.StatusForbidden, "users present")
}
