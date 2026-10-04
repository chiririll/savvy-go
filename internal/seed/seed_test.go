package seed

import (
	"context"
	"testing"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/domain"
	"savvy-go/internal/store"
	"savvy-go/internal/store/sqlite"
)

func openStore(t *testing.T) *sqlite.Store {
	t.Helper()
	st, err := sqlite.OpenApp(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// demoSpace is the database of the only space, the seeded demo space.
func demoSpace(t *testing.T, st *sqlite.Store) store.DB {
	t.Helper()
	ids, _ := st.Spaces(context.Background())
	if len(ids) != 1 {
		t.Fatalf("spaces %v, want only the demo space", ids)
	}
	d, err := st.Space(context.Background(), ids[0])
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
	if err := Demo(ctx, st, false, time.UTC); err != nil {
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
	st := openStore(t)
	sqlDB := st.Server()
	ctx := context.Background()
	if err := Demo(ctx, st, true, time.UTC); err != nil {
		t.Fatal(err)
	}

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
	spaces := domain.Spaces{Store: st}
	ids, _ := st.Spaces(ctx)
	for u, want := range map[*auth.User]string{admin: domain.SpaceAdmin, editor: domain.SpaceEditor, demo: domain.SpaceViewer} {
		if role, _ := spaces.Role(ctx, ids[0], u.ID); role != want {
			t.Fatalf("%s has role %q in the demo space, want %q", u.Email, role, want)
		}
	}
	space := demoSpace(t, st)

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

func TestDemoDoesNotReseed(t *testing.T) {
	st := openStore(t)
	sqlDB := st.Server()
	ctx := context.Background()
	if err := Demo(ctx, st, true, time.UTC); err != nil {
		t.Fatal(err)
	}
	space := demoSpace(t, st)
	users := countWhere(t, sqlDB, `SELECT COUNT(*) FROM users`)
	accts := countWhere(t, space, `SELECT COUNT(*) FROM accounts`)
	txs := countWhere(t, space, `SELECT COUNT(*) FROM transactions`)
	if err := Demo(ctx, st, true, time.UTC); err != nil {
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
	if err := Demo(ctx, st, true, time.UTC); err != nil {
		t.Fatal(err)
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM users WHERE email = 'demo@demo.com'`); n != 0 {
		t.Fatal("seeded over an existing install")
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM users`); n != 1 {
		t.Fatalf("users=%d", n)
	}
}
