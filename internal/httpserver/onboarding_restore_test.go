package httpserver

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"savvy-go/internal/auth"
	"savvy-go/internal/firstrun"
)

// firstRunRestore posts the restore form with no session, as the setup page does.
func (a *testApp) firstRunRestore(password, file string, content []byte, fields map[string]string) *http.Response {
	a.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if password != "" {
		_ = mw.WriteField("bootstrap_password", password)
	}
	if file != "" {
		fw, _ := mw.CreateFormFile("file", file)
		_, _ = fw.Write(content)
	}
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	_ = mw.Close()
	req, _ := http.NewRequest("POST", a.srv.URL+"/api/onboarding/restore", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := a.client.Do(req)
	if err != nil {
		a.t.Fatal(err)
	}
	return res
}

// serverBackupFile is a server backup of a server that has one admin, ada.
func serverBackupFile(t *testing.T) []byte {
	t.Helper()
	src := newTestApp(t)
	src.createUser("ada@source.com", "secret1", auth.RoleAdmin)
	entry, err := src.svc.Backups.CreateServer(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(src.svc.Backups.ServerPath(entry.Filename))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func userCount(t *testing.T, a *testApp) int {
	t.Helper()
	var n int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestFirstRunRestoreNeedsTheBootstrapPassword(t *testing.T) {
	a := newTestApp(t)
	content := serverBackupFile(t)

	fieldErrors(t, a.firstRunRestore("", "b.zip", content, nil), "bootstrap_password")
	fieldErrors(t, a.firstRunRestore("AAAAA-AAAAA-AAAAA-AAAAA-AAAAA-AAAAA", "b.zip", content, nil), "bootstrap_password")
	if n := userCount(t, a); n != 0 || !a.bootstrap.Active() {
		t.Fatalf("nothing may change without the password (users=%d)", n)
	}
}

func TestFirstRunRestoreIsRefusedOnceAUserExists(t *testing.T) {
	a := newTestApp(t)
	pass := a.bootstrap.Password()
	a.createUser("someone@test.com", "secret1", auth.RoleUser)
	status(t, a.firstRunRestore(pass, "b.zip", serverBackupFile(t), nil), http.StatusForbidden, "users present")
}

func TestFirstRunRestoreWithoutAGuardIsClosed(t *testing.T) {
	a := appWithGuard(t, nil)
	status(t, a.firstRunRestore("anything", "b.zip", serverBackupFile(t), nil), http.StatusForbidden, "no guard")
}

func TestFailedFirstRunRestoreKeepsThePasswordValid(t *testing.T) {
	a := newTestApp(t)
	pass := a.bootstrap.Password()

	res := a.firstRunRestore(pass, "b.zip", []byte("not an archive"), nil)
	status(t, res, http.StatusUnprocessableEntity, "bad archive")

	laravel, _ := os.ReadFile(laravelFile(t, true))
	res = a.firstRunRestore(pass, "laravel.sqlite", laravel, map[string]string{"app_key": "base64:wrong"})
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("wrong app key: %d", res.StatusCode)
	}
	res.Body.Close()

	status(t, a.firstRunRestore(pass, "", nil, nil), http.StatusUnprocessableEntity, "no file")

	if !a.bootstrap.Active() || a.bootstrap.Password() != pass {
		t.Fatal("a failed restore spent the password")
	}
	if n := userCount(t, a); n != 0 {
		t.Fatalf("%d users after failed restores", n)
	}
	if list, _ := a.svc.Backups.ServerBackups(); len(list) != 0 {
		t.Fatalf("failed restores left backups behind: %+v", list)
	}
}

func TestFirstRunRestoreLoadsTheBackupAndClosesSetup(t *testing.T) {
	a := newTestApp(t)
	pass := a.bootstrap.Password()

	res := a.firstRunRestore(pass, "b.zip", serverBackupFile(t), nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("restore %d %v", res.StatusCode, decodeJSON(t, res))
	}
	res.Body.Close()

	if u, _ := a.svc.Users.ByEmail(context.Background(), "ada@source.com"); u == nil || u.Role != auth.RoleAdmin {
		t.Fatalf("restored admin %+v", u)
	}
	if a.bootstrap.Active() {
		t.Fatal("password still valid after the restore")
	}
	if _, err := os.Stat(filepath.Join(a.svc.Cfg.DataDir, firstrun.FileName)); !os.IsNotExist(err) {
		t.Fatalf("password file still there: %v", err)
	}
	body := decodeJSON(t, a.do("GET", "/api/auth/status", nil, "", ""))
	if body["needs_registration"] != false || body["bootstrap_hint"] != nil {
		t.Fatalf("status after restore %v", body)
	}
	// The restored admin logs in with the password they had on the old server.
	status(
		t,
		a.do("POST", "/api/auth/login", map[string]any{"email": "ada@source.com", "password": "secret1"}, "", ""),
		http.StatusOK,
		"login",
	)
	// And the endpoint is gone with the setup.
	status(t, a.firstRunRestore(pass, "b.zip", serverBackupFile(t), nil), http.StatusForbidden, "second restore")
}

func TestFirstRunRestoreOfALaravelDatabase(t *testing.T) {
	a := newTestApp(t)
	pass := a.bootstrap.Password()
	laravel, _ := os.ReadFile(laravelFile(t, true))

	res := a.firstRunRestore(pass, "laravel.sqlite", laravel, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("restore %d %v", res.StatusCode, decodeJSON(t, res))
	}
	res.Body.Close()
	if u, _ := a.svc.Users.ByEmail(context.Background(), "ada@example.com"); u == nil || u.Role != auth.RoleAdmin {
		t.Fatalf("laravel admin %+v", u)
	}
	if a.bootstrap.Active() {
		t.Fatal("password still valid")
	}
}

func TestFirstRunRestoreWithoutUsersKeepsSetupOpen(t *testing.T) {
	a := newTestApp(t)
	pass := a.bootstrap.Password()
	empty := newTestApp(t)
	entry, err := empty.svc.Backups.CreateServer(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(empty.svc.Backups.ServerPath(entry.Filename))

	status(t, a.firstRunRestore(pass, "b.zip", raw, nil), http.StatusUnprocessableEntity, "backup without users")
	if !a.bootstrap.Active() {
		t.Fatal("a restore that leaves no users must keep the password: nobody could log in")
	}
}
