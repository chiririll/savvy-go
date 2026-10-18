package httpserver

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"savvy-go/internal/auth"
	"savvy-go/internal/auth/totp"
	"savvy-go/internal/db"
	"savvy-go/internal/ledger"
	"savvy-go/internal/legacy"
	"savvy-go/internal/spaces"
)

// upgradeLaravelServer uploads a Laravel database for the whole server and
// returns the name of the backup it was saved as.
func upgradeLaravelServer(
	t *testing.T,
	a *testApp,
	sess *auth.Issued,
	content []byte,
	fields map[string]string,
) string {
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
func TestLaravelDatabaseNotAcceptedAsBackup(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	content, _ := os.ReadFile(laravelFile(t, true))
	status(
		t,
		a.upload("/api/backups/upload", "laravel.sqlite", content, nil, sess),
		http.StatusUnprocessableEntity,
		"server upload",
	)
	status(
		t,
		a.upload(spacePath(a.space.ID, "/backups/upload"), "laravel.sqlite", content, nil, sess),
		http.StatusUnprocessableEntity,
		"space upload",
	)
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
	secret, err := totp.NewSecret()
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
	if _,
		err := d.Exec(`UPDATE users SET two_factor_secret = ?, two_factor_enabled = 1, two_factor_confirmed = 1 WHERE email `+
		`= 'ada@example.com'`, ct); err != nil {
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
func TestRestoreLaravelDecryptsTwoFactor(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	content, appKey, secret := laravelFileWithTOTP(t)

	status(
		t,
		a.upload("/api/backups/upgrade-laravel", "laravel.sqlite", content, map[string]string{"app_key": "nonsense"}, sess),
		http.StatusUnprocessableEntity,
		"invalid key",
	)
	name := upgradeLaravelServer(t, a, sess, content, map[string]string{"app_key": appKey})
	status(t, a.do("POST", "/api/backups/"+name+"/restore", nil, sess.Token, sess.CSRF), http.StatusOK, "restore")

	ada, err := a.svc.Users.ByEmail(context.Background(), "ada@example.com")
	if err != nil || ada == nil || ada.TwoFactorSecret == nil || *ada.TwoFactorSecret != secret {
		t.Fatalf("two-factor secret after restore = %+v, %v; want the decrypted secret", ada, err)
	}
}

// Without the key the secrets cannot be read, so they are reset and the user
// can still sign in with the password.
func TestRestoreLaravelKeylessResetsTwoFactor(t *testing.T) {
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
	if len(list) != 1 || list[0].Filename != name || !list[0].Valid || list[0].Kind != "space" ||
		list[0].SpaceUUID != before.UUID {
		t.Fatalf("saved backups %+v; want the upgraded one, of this space", list)
	}
	if n := len(decodeJSON(t,
		a.inSpace(a.space.ID, "GET", "/api/accounts", nil, sess))["data"].([]any)); n != accountsBefore {
		t.Fatalf("the upgrade changed the live space: %d accounts, had %d", n, accountsBefore)
	}

	status(
		t,
		a.do("POST", spacePath(a.space.ID, "/backups/"+name+"/restore"), nil, sess.Token, sess.CSRF),
		http.StatusOK,
		"restore",
	)

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

func loadLaravelFixture(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	script, err := os.ReadFile(filepath.Join("..", "..", "testdata", "laravel.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(string(script)); err != nil {
		t.Fatalf("load laravel.sql: %v", err)
	}
	if _,
		err := sqlDB.Exec(`INSERT INTO migrations (migration, batch) VALUES (?, 2)`, legacy.LatestMigration); err != nil {
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
