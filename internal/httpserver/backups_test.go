//go:build integration

package httpserver

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"savvy-go/internal/auth"
)

// upload posts a file as multipart "file" with extra form fields.
func (a *testApp) upload(
	path, file string,
	content []byte,
	fields map[string]string,
	sess *auth.Issued,
) *http.Response {
	a.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", file)
	_, _ = fw.Write(content)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	_ = mw.Close()
	req, _ := http.NewRequest("POST", a.srv.URL+path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "svy_session", Value: sess.Token})
	req.Header.Set("X-CSRF-Token", sess.CSRF)
	res, err := a.client.Do(req)
	if err != nil {
		a.t.Fatal(err)
	}
	return res
}

func spacePath(id int64, rest string) string {
	return "/api/spaces/" + strconv.FormatInt(id, 10) + rest
}

func createdBackup(t *testing.T, res *http.Response) map[string]any {
	t.Helper()
	body := decodeJSON(t, res)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create backup %d %v", res.StatusCode, body)
	}
	return body["data"].(map[string]any)
}

func tagCount(t *testing.T, a *testApp, spaceID int64, sess *auth.Issued) int {
	t.Helper()
	return len(tagNames(t, a.inSpace(spaceID, "GET", "/api/tags", nil, sess)))
}

// Space backups are for the space's admins only.
func TestSpaceBackupsAreForSpaceAdmins(t *testing.T) {
	a := newTestApp(t)
	for _, role := range []string{roleEditor, roleViewer} {
		sess := a.issue(a.createUser(role+"@test.com", "secret1", role), false)
		status(t, a.do("GET", spacePath(a.space.ID, "/backups"), nil, sess.Token, ""), 403, role+" list")
		status(
			t,
			a.do("POST", spacePath(a.space.ID, "/backups"), map[string]any{}, sess.Token, sess.CSRF),
			403,
			role+" create",
		)
	}
	admin := a.createUser("admin@test.com", "secret1", auth.RoleAdmin)
	raw, _ := a.issueToken(admin, tokReadWrite, nil)
	status(t, a.bearer("GET", spacePath(a.space.ID, "/backups"), nil, raw), 403, "API token")
	outsider := a.issue(a.createUser("out@test.com", "secret1", auth.RoleUser), false)
	status(t, a.do("GET", spacePath(a.space.ID, "/backups"), nil, outsider.Token, ""), 404, "not a member")
}

// A space admin backs a space up and restores it.
func TestSpaceBackupRestore(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	status(t, a.inSpace(a.space.ID, "POST", "/api/tags", map[string]any{"name": "before"}, sess), 201, "tag")
	b := createdBackup(
		t,
		a.do("POST", spacePath(a.space.ID, "/backups"), map[string]any{"note": "nightly"}, sess.Token, sess.CSRF),
	)
	if b["status"] != "current" || b["signature"] != "own" || b["kind"] != "space" || b["note"] != "nightly" {
		t.Fatalf("backup %v", b)
	}
	status(t, a.inSpace(a.space.ID, "POST", "/api/tags", map[string]any{"name": "after"}, sess), 201, "tag")
	if n := tagCount(t, a, a.space.ID, sess); n != 2 {
		t.Fatalf("tags %d", n)
	}
	name := b["filename"].(string)
	res := a.do("POST", spacePath(a.space.ID, "/backups/"+name+"/restore"), nil, sess.Token, sess.CSRF)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("restore %d %v", res.StatusCode, decodeJSON(t, res))
	}
	res.Body.Close()
	if n := tagCount(t, a, a.space.ID, sess); n != 1 {
		t.Fatalf("tags after restore %d", n)
	}

	res = a.do("GET", spacePath(a.space.ID, "/backups/"+name+"/download"), nil, sess.Token, "")
	raw, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || !bytes.HasPrefix(raw, []byte("PK")) {
		t.Fatalf("download %d", res.StatusCode)
	}
	status(t, a.do("DELETE", spacePath(a.space.ID, "/backups/"+name), nil, sess.Token, sess.CSRF), 204, "delete")
	status(
		t,
		a.do("GET", spacePath(a.space.ID, "/backups/../server/x.zip/download"), nil, sess.Token, ""),
		404,
		"path traversal",
	)
}

// Space backups beyond space_backups_max are removed, oldest first.
func TestSpaceBackupRetention(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	status(t, a.do("PATCH", "/api/settings", map[string]any{"space_backups_max": 2}, sess.Token, sess.CSRF), 200, "limit")
	for i := 0; i < 3; i++ {
		createdBackup(t, a.do("POST", spacePath(a.space.ID, "/backups"), map[string]any{}, sess.Token, sess.CSRF))
	}
	body := decodeJSON(t, a.do("GET", spacePath(a.space.ID, "/backups"), nil, sess.Token, ""))
	if n := len(body["data"].([]any)); n != 2 {
		t.Fatalf("kept %d backups", n)
	}
}

// A space backup cannot overwrite another space, even when the admin
// of both moves the file between them.
func TestBackupOfAnotherSpaceIsRefused(t *testing.T) {
	a := newTestApp(t)
	ctx := context.Background()
	admin := a.createUser("admin@test.com", "secret1", auth.RoleAdmin)
	sess := a.issue(admin, false)
	other, err := a.svc.Spaces.Create(ctx, "Other", admin)
	if err != nil {
		t.Fatal(err)
	}
	b := createdBackup(t, a.do("POST", spacePath(a.space.ID, "/backups"), map[string]any{}, sess.Token, sess.CSRF))
	content, _ := os.ReadFile(a.svc.Backups.SpacePath(a.space.ID, b["filename"].(string)))
	up := createdBackup(t, a.upload(spacePath(other.ID, "/backups/upload"), "x.zip", content, nil, sess))
	res := a.do("POST", spacePath(other.ID, "/backups/"+up["filename"].(string)+"/restore"), nil, sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body["message"].(string), "another space") {
		t.Fatalf("restore over another space %d %v", res.StatusCode, body)
	}
}

// A backup larger than the space quota is refused before it replaces
// anything.
func TestBackupOverQuotaIsRefused(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	for i := 0; i < 50; i++ {
		status(
			t,
			a.inSpace(a.space.ID, "POST", "/api/tags", map[string]any{"name": strings.Repeat("t", 200) + strconv.Itoa(i)}, sess),
			201,
			"tag",
		)
	}
	b := createdBackup(t, a.do("POST", spacePath(a.space.ID, "/backups"), map[string]any{}, sess.Token, sess.CSRF))
	_ = a.svc.Settings.Set(context.Background(), "space_quota_mb", 0.01)
	res := a.do("POST", spacePath(a.space.ID, "/backups/"+b["filename"].(string)+"/restore"), nil, sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("over quota %d %v", res.StatusCode, body)
	}
	if n := tagCount(t, a, a.space.ID, sess); n != 50 {
		t.Fatalf("space changed: %d tags", n)
	}
}

// The signing key travels in server backups only.
func TestSigningKeyTravelsOnlyServerBackups(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	names := func(path string) []string {
		zr, err := zip.OpenReader(path)
		if err != nil {
			t.Fatal(err)
		}
		defer zr.Close()
		var out []string
		for _, f := range zr.File {
			out = append(out, f.Name)
		}
		return out
	}
	sb := createdBackup(t, a.do("POST", spacePath(a.space.ID, "/backups"), map[string]any{}, sess.Token, sess.CSRF))
	for _, n := range names(a.svc.Backups.SpacePath(a.space.ID, sb["filename"].(string))) {
		if strings.HasPrefix(n, "keys/") {
			t.Fatalf("space backup holds %s", n)
		}
	}
	srv := createdBackup(t, a.do("POST", "/api/backups", map[string]any{}, sess.Token, sess.CSRF))
	if !strings.Contains(
		strings.Join(names(a.svc.Backups.ServerPath(srv["filename"].(string))), " "),
		"keys/server.ed25519",
	) {
		t.Fatal("server backup lacks the signing key")
	}
}

// A server backup restores the server and every space.
func TestServerBackupRestore(t *testing.T) {
	a := newTestApp(t)
	admin := a.createUser("admin@test.com", "secret1", auth.RoleAdmin)
	sess := a.issue(admin, false)
	status(t, a.inSpace(a.space.ID, "POST", "/api/tags", map[string]any{"name": "kept"}, sess), 201, "tag")
	b := createdBackup(t, a.do("POST", "/api/backups", map[string]any{}, sess.Token, sess.CSRF))
	if b["kind"] != "server" || b["status"] != "current" {
		t.Fatalf("server backup %v", b)
	}
	status(t, a.inSpace(a.space.ID, "POST", "/api/tags", map[string]any{"name": "lost"}, sess), 201, "tag")
	a.createUser("later@test.com", "secret1", auth.RoleUser)

	res := a.do("POST", "/api/backups/"+b["filename"].(string)+"/restore", nil, sess.Token, sess.CSRF)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("restore %d %v", res.StatusCode, decodeJSON(t, res))
	}
	res.Body.Close()
	if u, _ := a.svc.Users.ByEmail(context.Background(), "later@test.com"); u != nil {
		t.Fatal("a user created after the backup survived the restore")
	}
	if n := tagCount(t, a, a.space.ID, sess); n != 1 {
		t.Fatalf("tags after restore %d", n)
	}
}

// Backups hold every space, so only server admins reach them; space
// admins, editors, viewers and API tokens are refused.
func TestServerBackupsAreForServerAdmins(t *testing.T) {
	a := newTestApp(t)
	for _, role := range []string{roleEditor, roleViewer} {
		u := a.createUser(role+"@test.com", "secret1", role)
		sess := a.issue(u, false)
		status(t, a.do("GET", "/api/backups", nil, sess.Token, ""), 403, role+" list")
		status(t, a.do("POST", "/api/backups", map[string]any{}, sess.Token, sess.CSRF), 403, role+" create")
		status(t, a.do("POST", "/api/backups/x.sqlite/restore", nil, sess.Token, sess.CSRF), 403, role+" restore")
	}
	admin := a.createUser("admin@test.com", "secret1", auth.RoleAdmin)
	raw, _ := a.issueToken(admin, tokReadWrite, nil)
	status(t, a.bearer("GET", "/api/backups", nil, raw), 403, "admin API token")
}
