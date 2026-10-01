package seed

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/db"
	"savvy-go/internal/migrate"
)

func openMigrated(t *testing.T) *sql.DB {
	t.Helper()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "database.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := migrate.Up(context.Background(), sqlDB); err != nil {
		t.Fatal(err)
	}
	return sqlDB
}

func countWhere(t *testing.T, sqlDB *sql.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := sqlDB.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestDemoSkippedWhenDisabled(t *testing.T) {
	sqlDB := openMigrated(t)
	ctx := context.Background()
	if err := Demo(ctx, sqlDB, false, time.UTC); err != nil {
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
	sqlDB := openMigrated(t)
	ctx := context.Background()
	if err := Demo(ctx, sqlDB, true, time.UTC); err != nil {
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
	if demo.Role != auth.RoleReadOnly || admin.Role != auth.RoleAdmin {
		t.Fatalf("roles demo=%s admin=%s", demo.Role, admin.Role)
	}

	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM currencies WHERE code IN ('USD','EUR')`); n != 2 {
		t.Fatalf("currencies=%d", n)
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM categories`); n < 10 {
		t.Fatalf("categories=%d", n)
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM tags`); n < 8 {
		t.Fatalf("tags=%d", n)
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM accounts`); n < 8 {
		t.Fatalf("accounts=%d", n)
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM transactions`); n < 100 {
		t.Fatalf("transactions=%d", n)
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM budgets`); n < 4 {
		t.Fatalf("budgets=%d", n)
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM recurring_transactions`); n < 4 {
		t.Fatalf("recurring=%d", n)
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM automation_rules`); n < 2 {
		t.Fatalf("automation=%d", n)
	}
}

func TestDemoDoesNotReseed(t *testing.T) {
	sqlDB := openMigrated(t)
	ctx := context.Background()
	if err := Demo(ctx, sqlDB, true, time.UTC); err != nil {
		t.Fatal(err)
	}
	users := countWhere(t, sqlDB, `SELECT COUNT(*) FROM users`)
	accts := countWhere(t, sqlDB, `SELECT COUNT(*) FROM accounts`)
	txs := countWhere(t, sqlDB, `SELECT COUNT(*) FROM transactions`)
	if err := Demo(ctx, sqlDB, true, time.UTC); err != nil {
		t.Fatal(err)
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM users`); n != users {
		t.Fatalf("users %d -> %d", users, n)
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM accounts`); n != accts {
		t.Fatalf("accounts %d -> %d", accts, n)
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM transactions`); n != txs {
		t.Fatalf("transactions %d -> %d", txs, n)
	}
}

func TestDemoSkipsWhenUsersExist(t *testing.T) {
	sqlDB := openMigrated(t)
	ctx := context.Background()
	pass := "secret1"
	if _, err := (auth.Users{DB: sqlDB}).Create(ctx, "Owner", "owner@example.com", &pass, auth.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if err := Demo(ctx, sqlDB, true, time.UTC); err != nil {
		t.Fatal(err)
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM users WHERE email = 'demo@demo.com'`); n != 0 {
		t.Fatal("seeded over an existing install")
	}
	if n := countWhere(t, sqlDB, `SELECT COUNT(*) FROM users`); n != 1 {
		t.Fatalf("users=%d", n)
	}
}
