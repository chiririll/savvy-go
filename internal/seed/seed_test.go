package seed

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/auth/password"
	"savvy-go/internal/signing"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
	"savvy-go/internal/store/sqlite"
	"savvy-go/internal/transfers"
)

func openStore(t *testing.T) *sqlite.Store {
	t.Helper()
	st, _ := openStoreKeys(t)
	return st
}

func openStoreKeys(t *testing.T) (*sqlite.Store, signing.KeyRing) {
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
	return st, signing.KeyRing{Holder: signing.NewHolder(key), Trusted: transfers.TrustedKeys(st.Server())}
}

// seedDemo seeds a fresh store and returns it with its keys.
func seedDemo(t *testing.T) (*sqlite.Store, signing.KeyRing) {
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
	if err := st.Server().QueryRowContext(context.Background(), `SELECT id FROM spaces WHERE name = ?`,
		name).Scan(&id); err != nil {
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
	if err := Run(ctx, st, signing.KeyRing{}, Config{}, time.UTC); err != nil {
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
	if !password.Check(*demo.Password, "demo") {
		t.Fatal("demo password rejected")
	}
	admin, err := users.ByEmail(ctx, "admin@savvy.app")
	if err != nil || admin == nil || admin.Password == nil {
		t.Fatalf("admin user: %v %v", admin, err)
	}
	if !password.Check(*admin.Password, "password") {
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
	sps := spaces.Service{Store: st}
	for _, c := range []struct {
		space string
		user  *auth.User
		role  string
	}{
		{"Demo", admin, spaces.RoleAdmin}, {"Demo", editor, spaces.RoleEditor}, {"Demo", demo, spaces.RoleViewer},
		{"Family Budget", admin, spaces.RoleAdmin}, {"Family Budget", editor, spaces.RoleAdmin},
		{"Family Budget", guest, spaces.RoleViewer},
		{"Studio GmbH", admin, spaces.RoleAdmin}, {"Studio GmbH", editor, spaces.RoleViewer},
	} {
		var id int64
		if err := sqlDB.QueryRowContext(ctx, `SELECT id FROM spaces WHERE name = ?`, c.space).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if role, _ := sps.Role(ctx, id, c.user.ID); role != c.role {
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
	if n := countWhere(t, space, `SELECT COUNT(*) FROM tags`); n < 5 {
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

func TestDemoSeedsExactlyTheExportedPresets(t *testing.T) {
	st, _ := seedDemo(t)
	space := spaceByName(t, st, "Demo")
	cats, tags := DefaultCategories(), DefaultTags()
	if len(cats) != 20 || len(tags) != 5 {
		t.Fatalf("presets: %d categories, %d tags", len(cats), len(tags))
	}
	if n := countWhere(t, space, `SELECT COUNT(*) FROM categories`); n < len(cats) {
		t.Fatalf("categories=%d, want at least %d", n, len(cats))
	}
	for _, c := range cats {
		q := `SELECT COUNT(*) FROM categories WHERE name = '` + c.Name + `' AND type = '` + c.Type + `' AND icon = '` +
			c.Icon + `' AND color = '` + c.Color + `'`
		if n := countWhere(t, space, q); n != 1 {
			t.Fatalf("category %s seeded %d times", c.Name, n)
		}
	}
	for _, tag := range tags {
		if n := countWhere(t, space, `SELECT COUNT(*) FROM tags WHERE name = '`+tag+`'`); n != 1 {
			t.Fatalf("tag %s seeded %d times", tag, n)
		}
	}
	if n := countWhere(t, space, `SELECT COUNT(*) FROM categories WHERE is_default = 1`); n != 2 {
		t.Fatalf("default categories=%d", n)
	}
}

func TestPresetsAreCopies(t *testing.T) {
	DefaultCategories()[0].Name = "changed"
	DefaultTags()[0] = "changed"
	if DefaultCategories()[0].Name != "#HOUSING" || DefaultTags()[0] != "#ESSENTIAL" {
		t.Fatal("a caller changed the shared presets")
	}
}

func TestDemoSpacesAreLinkedTransfer(t *testing.T) {
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
	if _, err := Demo(ctx, st, signing.KeyRing{}, Options{Loc: time.UTC}); err != nil {
		t.Fatal(err)
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM users WHERE email = 'demo@demo.com'`); n != 0 {
		t.Fatal("seeded over an existing install")
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM users`); n != 1 {
		t.Fatalf("users=%d", n)
	}
}

func TestDemoPinnedDateReproducibleHasManifest(t *testing.T) {
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
		if err := space.QueryRowContext(context.Background(),
			`SELECT MAX(date) FROM transactions WHERE status = 'confirmed'`).Scan(&last); err != nil {
			t.Fatal(err)
		}
		if last > "2026-03-15" {
			t.Fatalf("confirmed transaction after the pinned date: %s", last)
		}
		var sum string
		if err := space.QueryRowContext(context.Background(),
			`SELECT COUNT(*) || ':' || SUM(amount) FROM transactions`).Scan(&sum); err != nil {
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
	if len(manifest.Spaces[0].Automations) == 0 || manifest.Spaces[2].Members["alex"] != spaces.RoleAdmin {
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

func TestDemoSpacesHaveLookAndAccount(t *testing.T) {
	st, _ := seedDemo(t)
	rows, err := st.Server().QueryContext(context.Background(), `SELECT name, emoji, color FROM spaces`)
	if err != nil {
		t.Fatal(err)
	}
	type look struct{ name, emoji, color string }
	var looks []look
	for rows.Next() {
		var l look
		var emoji, color *string
		if err := rows.Scan(&l.name, &emoji, &color); err != nil {
			t.Fatal(err)
		}
		if emoji == nil || color == nil {
			t.Fatalf("%s: emoji=%v color=%v", l.name, emoji, color)
		}
		l.emoji, l.color = *emoji, *color
		looks = append(looks, l)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	if len(looks) != 3 {
		t.Fatalf("spaces=%d", len(looks))
	}
	seen := map[string]bool{}
	for _, l := range looks {
		if seen[l.emoji] || seen[l.color] {
			t.Fatalf("%s repeats the emoji or color of another space", l.name)
		}
		seen[l.emoji], seen[l.color] = true, true
		if c := countWhere(t, spaceByName(t, st, l.name), `SELECT COUNT(*) FROM accounts`); c == 0 {
			t.Fatalf("%s has no account", l.name)
		}
	}
}

func TestDemoSyncGroup(t *testing.T) {
	st, keys := openStoreKeys(t)
	m, err := Demo(context.Background(), st, keys, Options{Loc: time.UTC})
	if err != nil {
		t.Fatal(err)
	}
	if n := countWhere(t, st.Server(), `SELECT COUNT(*) FROM sync_groups`); n != 1 {
		t.Fatalf("sync groups=%d", n)
	}
	group := map[string]int64{}
	for _, sp := range m.Spaces {
		group[sp.Name] = sp.SyncGroup
	}
	if group["Family Budget"] == 0 || group["Family Budget"] != group["Studio GmbH"] || group["Demo"] != 0 {
		t.Fatalf("manifest sync groups: %v", group)
	}
	for _, name := range []string{"Family Budget", "Studio GmbH"} {
		d := spaceByName(t, st, name)
		for _, cat := range []string{"School & kids", "Software & tools"} {
			if countWhere(t, d, `SELECT COUNT(*) FROM categories WHERE name = ?`, cat) != 1 {
				t.Fatalf("%s lacks the shared category %q", name, cat)
			}
		}
		for _, tag := range []string{"Kids", "Client work"} {
			if countWhere(t, d, `SELECT COUNT(*) FROM tags WHERE name = ?`, tag) != 1 {
				t.Fatalf("%s lacks the shared tag %q", name, tag)
			}
		}
	}
	if countWhere(t, spaceByName(t, st, "Demo"), `SELECT COUNT(*) FROM categories WHERE name = 'School & kids'`) != 0 {
		t.Fatal("Demo got a category of the group")
	}
}

func TestDemoSamIsUnverified(t *testing.T) {
	st, keys := openStoreKeys(t)
	m, err := Demo(context.Background(), st, keys, Options{Loc: time.UTC})
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range m.Users {
		if u.Verified == (u.Key == userSam) {
			t.Fatalf("%s verified=%v", u.Key, u.Verified)
		}
		want := 1
		if !u.Verified {
			want = 0
		}
		if n := countWhere(t, st.Server(), `SELECT email_verified FROM users WHERE email = ?`, u.Email); n != want {
			t.Fatalf("%s email_verified=%d", u.Key, n)
		}
	}
}

func TestPresetsAreUniqueTokensOneDefaultEach(t *testing.T) {
	seen := map[string]bool{}
	defaults := map[string]int{}
	for _, c := range DefaultCategories() {
		if !strings.HasPrefix(c.Name, "#") {
			t.Fatalf("category %q is not a token", c.Name)
		}
		if seen[c.Type+c.Name] {
			t.Fatalf("category %s duplicated in %s", c.Name, c.Type)
		}
		seen[c.Type+c.Name] = true
		if c.IsDefault {
			defaults[c.Type]++
		}
	}
	if defaults["expense"] != 1 || defaults["income"] != 1 {
		t.Fatalf("defaults per type: %v", defaults)
	}
	tags := map[string]bool{}
	for _, tag := range DefaultTags() {
		if !strings.HasPrefix(tag, "#") || tags[tag] {
			t.Fatalf("tag %q is not a unique token", tag)
		}
		tags[tag] = true
	}
}
