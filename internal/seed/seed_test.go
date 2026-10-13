package seed

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/domain"
	"savvy-go/internal/signing"
	"savvy-go/internal/store"
	"savvy-go/internal/store/sqlite"
)

func openStore(t *testing.T) *sqlite.Store {
	t.Helper()
	st, _ := openStoreKeys(t)
	return st
}

func openStoreKeys(t *testing.T) (*sqlite.Store, domain.KeyRing) {
	t.Helper()
	dir := t.TempDir()
	st, err := sqlite.OpenApp(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	key, err := signing.Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	return st, domain.KeyRing{Holder: signing.NewHolder(key), Trusted: domain.TrustedKeys(st.Server())}
}

// seedDemo seeds a fresh store and returns it with its keys.
func seedDemo(t *testing.T) (*sqlite.Store, domain.KeyRing) {
	t.Helper()
	st, keys := openStoreKeys(t)
	if _, err := Demo(context.Background(), st, keys, Options{Loc: time.UTC}); err != nil {
		t.Fatal(err)
	}
	return st, keys
}

// spaceByName is the database of the seeded space called name.
func spaceByName(t *testing.T, st *sqlite.Store, name string) store.DB {
	t.Helper()
	var id int64
	if err := st.Server().QueryRowContext(context.Background(), `SELECT id FROM spaces WHERE name = ?`, name).Scan(&id); err != nil {
		t.Fatalf("space %q: %v", name, err)
	}
	d, err := st.Space(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func countWhere(t *testing.T, d store.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := d.QueryRowContext(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestDemoSkippedWhenDisabled(t *testing.T) {
	st := openStore(t)
	sqlDB := st.Server()
	ctx := context.Background()
	if err := Run(ctx, st, domain.KeyRing{}, Config{}, time.UTC); err != nil {
		t.Fatal(err)
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM users`); n != 0 {
		t.Fatalf("users=%d, want 0", n)
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM users WHERE email IN ('admin@savvy.app','demo@demo.com')`); n != 0 {
		t.Fatalf("demo users present with SEED_DEMO=false: %d", n)
	}
}

func TestDemoSeedsEmptyDatabase(t *testing.T) {
	st, _ := seedDemo(t)
	sqlDB := st.Server()
	ctx := context.Background()

	for _, email := range []string{"admin@savvy.app", "editor@savvy.app", "demo@demo.com"} {
		if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM users WHERE email = ?`, email); n != 1 {
			t.Fatalf("%s count=%d", email, n)
		}
	}

	users := auth.Users{DB: sqlDB}
	demo, err := users.ByEmail(ctx, "demo@demo.com")
	if err != nil || demo == nil || demo.Password == nil {
		t.Fatalf("demo user: %v %v", demo, err)
	}
	if !auth.CheckPassword(*demo.Password, "demo") {
		t.Fatal("demo password rejected")
	}
	admin, err := users.ByEmail(ctx, "admin@savvy.app")
	if err != nil || admin == nil || admin.Password == nil {
		t.Fatalf("admin user: %v %v", admin, err)
	}
	if !auth.CheckPassword(*admin.Password, "password") {
		t.Fatal("admin password rejected")
	}
	if demo.Role != auth.RoleUser || admin.Role != auth.RoleAdmin {
		t.Fatalf("roles demo=%s admin=%s", demo.Role, admin.Role)
	}
	editor, _ := users.ByEmail(ctx, "editor@savvy.app")
	guest, _ := users.ByEmail(ctx, "guest@savvy.app")
	if guest == nil || !guest.IsGuest() {
		t.Fatalf("guest user: %v", guest)
	}
	spaces := domain.Spaces{Store: st}
	for _, c := range []struct {
		space string
		user  *auth.User
		role  string
	}{
		{"Demo", admin, domain.SpaceAdmin}, {"Demo", editor, domain.SpaceEditor}, {"Demo", demo, domain.SpaceViewer},
		{"Family Budget", admin, domain.SpaceAdmin}, {"Family Budget", editor, domain.SpaceAdmin}, {"Family Budget", guest, domain.SpaceViewer},
		{"Studio GmbH", admin, domain.SpaceAdmin}, {"Studio GmbH", editor, domain.SpaceViewer},
	} {
		var id int64
		if err := sqlDB.QueryRowContext(ctx, `SELECT id FROM spaces WHERE name = ?`, c.space).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if role, _ := spaces.Role(ctx, id, c.user.ID); role != c.role {
			t.Fatalf("%s has role %q in %s, want %q", c.user.Email, role, c.space, c.role)
		}
	}
	space := spaceByName(t, st, "Demo")

	if n := countWhere(t, space, `SELECT COUNT(*) FROM currencies WHERE code IN ('USD','EUR')`); n != 2 {
		t.Fatalf("currencies=%d", n)
	}
	if n := countWhere(t, space, `SELECT COUNT(*) FROM categories`); n < 10 {
		t.Fatalf("categories=%d", n)
	}
	if n := countWhere(t, space, `SELECT COUNT(*) FROM tags`); n < 8 {
		t.Fatalf("tags=%d", n)
	}
	if n := countWhere(t, space, `SELECT COUNT(*) FROM accounts`); n < 8 {
		t.Fatalf("accounts=%d", n)
	}
	if n := countWhere(t, space, `SELECT COUNT(*) FROM transactions`); n < 100 {
		t.Fatalf("transactions=%d", n)
	}
	if n := countWhere(t, space, `SELECT COUNT(*) FROM budgets`); n < 4 {
		t.Fatalf("budgets=%d", n)
	}
	if n := countWhere(t, space, `SELECT COUNT(*) FROM recurring_transactions`); n < 4 {
		t.Fatalf("recurring=%d", n)
	}
	if n := countWhere(t, space, `SELECT COUNT(*) FROM automation_rules`); n < 2 {
		t.Fatalf("automation=%d", n)
	}
}

func TestDemoSpacesAreLinkedAndTransfer(t *testing.T) {
	st, _ := seedDemo(t)
	sqlDB := st.Server()
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM spaces`); n != 3 {
		t.Fatalf("spaces=%d", n)
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM space_links`); n != 2 {
		t.Fatalf("links=%d, want Demo linked to the two others", n)
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM space_invitations WHERE accepted_at IS NULL`); n != 2 {
		t.Fatalf("open invitations=%d", n)
	}
	for _, name := range []string{"Demo", "Family Budget", "Studio GmbH"} {
		d := spaceByName(t, st, name)
		if n := countWhere(t, d, `SELECT COUNT(*) FROM space_transfers`); n == 0 {
			t.Fatalf("%s has no transfers", name)
		}
		if n := countWhere(t, d, `SELECT COUNT(*) FROM transactions WHERE type IN ('transfer_in','transfer_out')`); n == 0 {
			t.Fatalf("%s has no transfer transactions", name)
		}
	}
}

func TestDemoHasEstimatedPending(t *testing.T) {
	st, _ := seedDemo(t)
	for _, name := range []string{"Demo", "Family Budget", "Studio GmbH"} {
		d := spaceByName(t, st, name)
		if n := countWhere(t, d, `SELECT COUNT(*) FROM transactions WHERE status = 'pending' AND is_estimated = 1`); n == 0 {
			t.Fatalf("%s has no pending transaction with an estimated amount", name)
		}
	}
}

func TestDemoDoesNotReseed(t *testing.T) {
	st, keys := seedDemo(t)
	sqlDB := st.Server()
	ctx := context.Background()
	space := spaceByName(t, st, "Demo")
	users := countWhere(t, sqlDB, `SELECT COUNT(*) FROM users`)
	accts := countWhere(t, space, `SELECT COUNT(*) FROM accounts`)
	txs := countWhere(t, space, `SELECT COUNT(*) FROM transactions`)
	if _, err := Demo(ctx, st, keys, Options{Loc: time.UTC}); err != nil {
		t.Fatal(err)
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM users`); n != users {
		t.Fatalf("users %d -> %d", users, n)
	}
	if n := countWhere(t, space, `SELECT COUNT(*) FROM accounts`); n != accts {
		t.Fatalf("accounts %d -> %d", accts, n)
	}
	if n := countWhere(t, space, `SELECT COUNT(*) FROM transactions`); n != txs {
		t.Fatalf("transactions %d -> %d", txs, n)
	}
}

func TestDemoSkipsWhenUsersExist(t *testing.T) {
	st := openStore(t)
	sqlDB := st.Server()
	ctx := context.Background()
	pass := "secret1"
	if _, err := (auth.Users{DB: sqlDB}).Create(ctx, "Owner", "owner@example.com", &pass, auth.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if _, err := Demo(ctx, st, domain.KeyRing{}, Options{Loc: time.UTC}); err != nil {
		t.Fatal(err)
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM users WHERE email = 'demo@demo.com'`); n != 0 {
		t.Fatal("seeded over an existing install")
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM users`); n != 1 {
		t.Fatalf("users=%d", n)
	}
}

func TestDemoPinnedDateIsReproducibleAndHasManifest(t *testing.T) {
	now, err := Config{Date: "2026-03-15"}.now(time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	var dumps [2]string
	var manifest *Manifest
	for i := range dumps {
		st, keys := openStoreKeys(t)
		manifest, err = Demo(context.Background(), st, keys, Options{Loc: time.UTC, Now: now})
		if err != nil {
			t.Fatal(err)
		}
		space := spaceByName(t, st, "Demo")
		var last string
		if err := space.QueryRowContext(context.Background(), `SELECT MAX(date) FROM transactions WHERE status = 'confirmed'`).Scan(&last); err != nil {
			t.Fatal(err)
		}
		if last > "2026-03-15" {
			t.Fatalf("confirmed transaction after the pinned date: %s", last)
		}
		var sum string
		if err := space.QueryRowContext(context.Background(), `SELECT COUNT(*) || ':' || SUM(amount) FROM transactions`).Scan(&sum); err != nil {
			t.Fatal(err)
		}
		dumps[i] = sum
	}
	if dumps[0] != dumps[1] {
		t.Fatalf("same date seeded different data: %s vs %s", dumps[0], dumps[1])
	}
	if manifest == nil || !manifest.Now.Equal(now) || len(manifest.Users) != 4 || len(manifest.Spaces) != 3 {
		t.Fatalf("manifest: %+v", manifest)
	}
	if len(manifest.Invitations) != 2 || manifest.Invitations[0].Token == "" {
		t.Fatalf("invitations: %+v", manifest.Invitations)
	}
	if len(manifest.Spaces[0].Automations) == 0 || manifest.Spaces[2].Members["alex"] != domain.SpaceAdmin {
		t.Fatalf("spaces: %+v", manifest.Spaces)
	}
}

func TestConfigRejectsBadDate(t *testing.T) {
	if _, err := (Config{Date: "15.03.2026"}).now(time.UTC); err == nil {
		t.Fatal("accepted a malformed date")
	}
}

func TestRunWritesManifest(t *testing.T) {
	st, keys := openStoreKeys(t)
	path := filepath.Join(t.TempDir(), "out", "manifest.json")
	cfg := Config{Enabled: true, Date: "2026-03-15", Manifest: path}
	if err := Run(context.Background(), st, keys, cfg, time.UTC); err != nil {
		t.Fatal(err)
	}
	var m Manifest
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if got := m.Now.Format("2006-01-02T15:04:05Z07:00"); got != "2026-03-15T12:00:00Z" {
		t.Fatalf("manifest now = %s", got)
	}
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "seed.toml")
	if err := os.WriteFile(good, []byte("date = \"2026-03-15\"\nmanifest = \"/m.json\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(good)
	if err != nil || !c.Enabled || c.Date != "2026-03-15" || c.Manifest != "/m.json" {
		t.Fatalf("got %+v, %v", c, err)
	}
	bad := filepath.Join(dir, "bad.toml")
	if err := os.WriteFile(bad, []byte("dat = \"x\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(bad); err == nil {
		t.Error("accepted an unknown key")
	}
}
