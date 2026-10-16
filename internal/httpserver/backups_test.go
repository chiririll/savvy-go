package httpserver

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"savvy-go/internal/auth"
	"savvy-go/internal/db"
	"savvy-go/internal/ledger"
	"savvy-go/internal/legacy"
	"savvy-go/internal/spaces"
)

func loadLaravelFixture(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	script, err := os.ReadFile(filepath.Join("..", "..", "testdata", "laravel.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(string(script)); err != nil {
		t.Fatalf("load laravel.sql: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO migrations (migration, batch) VALUES (?, 2)`, legacy.LatestMigration); err != nil {
		t.Fatal(err)
	}
}

// laravelFile writes a Laravel-era database; latest makes it importable.
func laravelFile(t *testing.T, latest bool) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "laravel.sqlite")
	d, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	loadLaravelFixture(t, d)
	if !latest {
		if _, err := d.Exec(`DELETE FROM migrations WHERE migration = ?`, legacy.LatestMigration); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := d.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatal(err)
	}
	_ = d.Close()
	return path
}

// upload posts a file as multipart "file" with extra form fields.
func (a *testApp) upload(path, file string, content []byte, fields map[string]string, sess *auth.Issued) *http.Response {
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
		status(t, a.do("POST", spacePath(a.space.ID, "/backups"), map[string]any{}, sess.Token, sess.CSRF), 403, role+" create")
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
	b := createdBackup(t, a.do("POST", spacePath(a.space.ID, "/backups"), map[string]any{"note": "nightly"}, sess.Token, sess.CSRF))
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
	status(t, a.do("GET", spacePath(a.space.ID, "/backups/../server/x.zip/download"), nil, sess.Token, ""), 404, "path traversal")
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

// rezip rewrites an archive, letting edit change or drop entries and add new
// ones.
func rezip(t *testing.T, src string, edit func(name string, data []byte) ([]byte, bool), extra map[string][]byte) []byte {
	t.Helper()
	zr, err := zip.OpenReader(src)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range zr.File {
		r, _ := f.Open()
		data, _ := io.ReadAll(r)
		r.Close()
		if data, keep := edit(f.Name, data); keep {
			w, _ := zw.Create(f.Name)
			_, _ = w.Write(data)
		}
	}
	for name, data := range extra {
		w, _ := zw.Create(name)
		_, _ = w.Write(data)
	}
	_ = zw.Close()
	return buf.Bytes()
}

// How a tampered archive shows through the API: an edited manifest
// is listed "unsigned" but restores, an inconsistent archive is a 422. The
// archive checks themselves are tested in internal/backup.
func TestTamperedArchives(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	b := createdBackup(t, a.do("POST", spacePath(a.space.ID, "/backups"), map[string]any{}, sess.Token, sess.CSRF))
	orig := a.svc.Backups.SpacePath(a.space.ID, b["filename"].(string))

	restore := func(content []byte) (int, map[string]any) {
		up := createdBackup(t, a.upload(spacePath(a.space.ID, "/backups/upload"), "x.zip", content, nil, sess))
		res := a.do("POST", spacePath(a.space.ID, "/backups/"+up["filename"].(string)+"/restore"), nil, sess.Token, sess.CSRF)
		body := decodeJSON(t, res)
		body["listed"] = up
		return res.StatusCode, body
	}

	// The note changed: still restorable, but no longer signed by us.
	edited := rezip(t, orig, func(name string, data []byte) ([]byte, bool) {
		if name == "manifest.json" {
			var m map[string]any
			_ = json.Unmarshal(data, &m)
			m["note"] = "edited"
			data, _ = json.Marshal(m)
		}
		return data, true
	}, nil)
	code, body := restore(edited)
	if listed := body["listed"].(map[string]any); listed["status"] != "unsigned" || code != http.StatusOK {
		t.Fatalf("edited manifest: listed %v restore %d %v", listed, code, body)
	}

	// The database changed: it no longer matches the manifest.
	swapped := rezip(t, orig, func(name string, data []byte) ([]byte, bool) {
		if name == "space.sqlite" {
			data = append(bytes.Clone(data), 0)
		}
		return data, true
	}, nil)
	if code, body := restore(swapped); code != http.StatusUnprocessableEntity || !strings.Contains(body["message"].(string), "does not match") {
		t.Fatalf("swapped database: %d %v", code, body)
	}

	bad := rezip(t, orig, func(name string, data []byte) ([]byte, bool) { return data, true }, map[string][]byte{"../escape.sqlite": []byte("x")})
	if code, body := restore(bad); code != http.StatusUnprocessableEntity {
		t.Fatalf("unexpected entry: %d %v", code, body)
	}
}

// A backup larger than the space quota is refused before it replaces
// anything.
func TestBackupOverQuotaIsRefused(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	for i := 0; i < 50; i++ {
		status(t, a.inSpace(a.space.ID, "POST", "/api/tags", map[string]any{"name": strings.Repeat("t", 200) + strconv.Itoa(i)}, sess), 201, "tag")
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
func TestSigningKeyTravelsOnlyInServerBackups(t *testing.T) {
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
	if !strings.Contains(strings.Join(names(a.svc.Backups.ServerPath(srv["filename"].(string))), " "), "keys/server.ed25519") {
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

// upgradeLaravelServer uploads a Laravel database for the whole server and
// returns the name of the backup it was saved as.
func upgradeLaravelServer(t *testing.T, a *testApp, sess *auth.Issued, content []byte, fields map[string]string) string {
	t.Helper()
	res := a.upload("/api/backups/upgrade-laravel", "laravel.sqlite", content, fields, sess)
	body := decodeJSON(t, res)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("upgrade %d %v", res.StatusCode, body)
	}
	return body["data"].(map[string]any)["filename"].(string)
}

// A Laravel database is upgraded and saved as a server backup, and the
// live data stays as it is until that backup is restored. Restoring it brings
// the Laravel admin, and its dates come back date-only.
func TestUpgradeLaravelBackupAsServer(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	content, _ := os.ReadFile(laravelFile(t, true))
	name := upgradeLaravelServer(t, a, sess, content, nil)

	list, _ := a.svc.Backups.ServerBackups()
	if len(list) != 1 || list[0].Filename != name || !list[0].Valid || list[0].Kind != "server" {
		t.Fatalf("saved backups %+v; want only the upgraded one", list)
	}
	if u, _ := a.svc.Users.ByEmail(context.Background(), "ada@example.com"); u != nil {
		t.Fatal("the upgrade changed live data")
	}
	if u, _ := a.svc.Users.ByEmail(context.Background(), "admin@test.com"); u == nil {
		t.Fatal("the upgrade removed a live user")
	}
	status(t, a.do("POST", "/api/backups/"+name+"/restore", nil, sess.Token, sess.CSRF), http.StatusOK, "restore")

	ctx := context.Background()
	ada, _ := a.svc.Users.ByEmail(ctx, "ada@example.com")
	if ada == nil || ada.Role != auth.RoleAdmin {
		t.Fatalf("laravel admin %+v", ada)
	}
	if role, _ := a.svc.Spaces.Role(ctx, 1, ada.ID); role != spaces.RoleAdmin {
		t.Fatalf("laravel admin's space role %q", role)
	}
	d, err := a.store.Space(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := (ledger.Transactions{DB: d}).ByID(ctx, 1)
	if err != nil || tx == nil || tx.Date == nil || *tx.Date != "2026-01-03" {
		t.Fatalf("restored transaction date = %+v, %v; want 2026-01-03", tx, err)
	}
}

// A Laravel database older than the supported version is refused and nothing
// is saved.
func TestUpgradeOldLaravelBackupSavesNothing(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("admin@test.com", "secret1", auth.RoleAdmin)
	sess := a.issue(u, false)
	content, _ := os.ReadFile(laravelFile(t, false))
	res := a.upload("/api/backups/upgrade-laravel", "old.sqlite", content, nil, sess)
	body := decodeJSON(t, res)
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("upgrade %d %v", res.StatusCode, body)
	}
	if list, _ := a.svc.Backups.ServerBackups(); len(list) != 0 {
		t.Fatalf("saved %v", list)
	}
	if got, _ := a.svc.Users.ByEmail(context.Background(), "admin@test.com"); got == nil {
		t.Fatal("live data damaged")
	}
}

// A Laravel backup imports as a new space owned by the caller, without
// its users; guests cannot import.
func TestImportLaravelBackupAsSpace(t *testing.T) {
	a := newTestApp(t)
	user := a.createUser("owner@test.com", "secret1", auth.RoleUser)
	sess := a.issue(user, false)
	content, _ := os.ReadFile(laravelFile(t, true))
	res := a.upload("/api/spaces/import", "laravel.sqlite", content, map[string]string{"name": "From Laravel"}, sess)
	body := decodeJSON(t, res)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("import %d %v", res.StatusCode, body)
	}
	data := body["data"].(map[string]any)
	id := int64(data["id"].(float64))
	if data["role"] != spaces.RoleAdmin || data["name"] != "From Laravel" {
		t.Fatalf("imported %v", data)
	}
	res = a.inSpace(id, "GET", "/api/accounts", nil, sess)
	accounts := decodeJSON(t, res)["data"].([]any)
	if len(accounts) == 0 {
		t.Fatal("imported space has no accounts")
	}
	if u, _ := a.svc.Users.ByEmail(context.Background(), "ada@example.com"); u != nil {
		t.Fatal("a space import brought the Laravel users")
	}

	guest := a.issue(a.createUser("guest@test.com", "secret1", auth.RoleGuest), false)
	status(t, a.upload("/api/spaces/import", "laravel.sqlite", content, nil, guest), 403, "guest import")
}

// Importing a backup next to its original gives the copy a new uuid, so
// the two are never taken for one space.
func TestImportCopyGetsNewUUID(t *testing.T) {
	a := newTestApp(t)
	admin := a.createUser("admin@test.com", "secret1", auth.RoleAdmin)
	sess := a.issue(admin, false)
	b := createdBackup(t, a.do("POST", spacePath(a.space.ID, "/backups"), map[string]any{}, sess.Token, sess.CSRF))
	content, _ := os.ReadFile(a.svc.Backups.SpacePath(a.space.ID, b["filename"].(string)))
	res := a.upload("/api/spaces/import", "copy.zip", content, nil, sess)
	body := decodeJSON(t, res)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("import %d %v", res.StatusCode, body)
	}
	copyUUID := body["data"].(map[string]any)["uuid"].(string)
	if copyUUID == a.space.UUID {
		t.Fatal("the copy kept the original's uuid")
	}
	if body["data"].(map[string]any)["name"] != "Test" {
		t.Fatalf("the copy should take the backup's space name: %v", body)
	}
}

// A Laravel database is not accepted by the plain backup upload: only its
// upgraded version is kept.
func TestLaravelDatabaseIsNotAcceptedAsBackup(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	content, _ := os.ReadFile(laravelFile(t, true))
	status(t, a.upload("/api/backups/upload", "laravel.sqlite", content, nil, sess), http.StatusUnprocessableEntity, "server upload")
	status(t, a.upload(spacePath(a.space.ID, "/backups/upload"), "laravel.sqlite", content, nil, sess), http.StatusUnprocessableEntity, "space upload")
	if list, _ := a.svc.Backups.ServerBackups(); len(list) != 0 {
		t.Fatalf("kept as a backup: %v", list)
	}
}

// laravelFileWithTOTP is a Laravel database whose admin has a two-factor
// secret, encrypted with a fresh APP_KEY.
func laravelFileWithTOTP(t *testing.T) (content []byte, appKey, secret string) {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	appKey = "base64:" + base64.StdEncoding.EncodeToString(key)
	secret, err := auth.NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	ct, err := legacy.EncryptLaravel(appKey, secret)
	if err != nil {
		t.Fatal(err)
	}
	path := laravelFile(t, true)
	d, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`UPDATE users SET two_factor_secret = ?, two_factor_enabled = 1, two_factor_confirmed = 1 WHERE email = 'ada@example.com'`, ct); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatal(err)
	}
	_ = d.Close()
	content, _ = os.ReadFile(path)
	return content, appKey, secret
}

// The Laravel APP_KEY entered at upload decrypts the two-factor secrets while
// the database is converted.
func TestRestoreLaravelServerDecryptsTwoFactorWithKey(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	content, appKey, secret := laravelFileWithTOTP(t)

	status(t, a.upload("/api/backups/upgrade-laravel", "laravel.sqlite", content, map[string]string{"app_key": "nonsense"}, sess), http.StatusUnprocessableEntity, "invalid key")
	name := upgradeLaravelServer(t, a, sess, content, map[string]string{"app_key": appKey})
	status(t, a.do("POST", "/api/backups/"+name+"/restore", nil, sess.Token, sess.CSRF), http.StatusOK, "restore")

	ada, err := a.svc.Users.ByEmail(context.Background(), "ada@example.com")
	if err != nil || ada == nil || ada.TwoFactorSecret == nil || *ada.TwoFactorSecret != secret {
		t.Fatalf("two-factor secret after restore = %+v, %v; want the decrypted secret", ada, err)
	}
}

// Without the key the secrets cannot be read, so they are reset and the user
// can still sign in with the password.
func TestRestoreLaravelServerWithoutKeyResetsTwoFactor(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	content, _, _ := laravelFileWithTOTP(t)

	name := upgradeLaravelServer(t, a, sess, content, nil)
	status(t, a.do("POST", "/api/backups/"+name+"/restore", nil, sess.Token, sess.CSRF), http.StatusOK, "restore")

	ada, err := a.svc.Users.ByEmail(context.Background(), "ada@example.com")
	if err != nil || ada == nil {
		t.Fatalf("laravel admin missing: %+v, %v", ada, err)
	}
	if ada.TwoFactorSecret != nil || ada.TwoFactorEnabled || ada.TwoFactorConfirmed {
		t.Fatalf("two-factor left on with an unreadable secret: %+v", ada)
	}
}

// A Laravel database is upgraded and saved as a backup of the current space,
// which stays as it is until the backup is restored. The restore keeps the
// space's identity.
func TestUpgradeLaravelIntoCurrentSpaceBackup(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	content, _ := os.ReadFile(laravelFile(t, true))
	before, _ := a.svc.Spaces.Get(context.Background(), a.space.ID)
	accountsBefore := len(decodeJSON(t, a.inSpace(a.space.ID, "GET", "/api/accounts", nil, sess))["data"].([]any))

	res := a.upload(spacePath(a.space.ID, "/backups/upgrade-laravel"), "laravel.sqlite", content, nil, sess)
	body := decodeJSON(t, res)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("upgrade %d %v", res.StatusCode, body)
	}
	name := body["data"].(map[string]any)["filename"].(string)
	list, _ := a.svc.Backups.SpaceBackups(a.space.ID)
	if len(list) != 1 || list[0].Filename != name || !list[0].Valid || list[0].Kind != "space" || list[0].SpaceUUID != before.UUID {
		t.Fatalf("saved backups %+v; want the upgraded one, of this space", list)
	}
	if n := len(decodeJSON(t, a.inSpace(a.space.ID, "GET", "/api/accounts", nil, sess))["data"].([]any)); n != accountsBefore {
		t.Fatalf("the upgrade changed the live space: %d accounts, had %d", n, accountsBefore)
	}

	status(t, a.do("POST", spacePath(a.space.ID, "/backups/"+name+"/restore"), nil, sess.Token, sess.CSRF), http.StatusOK, "restore")

	after, _ := a.svc.Spaces.Get(context.Background(), a.space.ID)
	if after == nil || before == nil || after.UUID != before.UUID {
		t.Fatalf("space identity changed: %+v -> %+v", before, after)
	}
	accounts := decodeJSON(t, a.inSpace(a.space.ID, "GET", "/api/accounts", nil, sess))["data"].([]any)
	if len(accounts) == 0 {
		t.Fatal("the space has none of the Laravel accounts")
	}
	if u, _ := a.svc.Users.ByEmail(context.Background(), "ada@example.com"); u != nil {
		t.Fatal("restoring into a space brought the Laravel users")
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
