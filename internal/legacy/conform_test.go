package legacy

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"savvy-go/internal/db"
	"savvy-go/internal/migrate"
)

// addRepeatedNames adds names that differ only by case, which the old schema
// allowed and the current one does not.
func addRepeatedNames(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	for _, q := range []string{
		`INSERT INTO categories (id, name, type) VALUES (10, 'groceries', 'expense')`,
		`INSERT INTO categories (id, name, type) VALUES (11, 'GROCERIES', 'expense')`,
		`INSERT INTO categories (id, name, type) VALUES (12, 'Groceries', 'income')`,
		`CREATE TABLE tags (id integer primary key autoincrement not null, name varchar not null, created_at datetime, updated_at datetime)`,
		`CREATE UNIQUE INDEX tags_name_unique ON tags (name)`,
		`INSERT INTO tags (id, name) VALUES (1, 'Food'), (2, 'food'), (3, 'Food (1)')`,
	} {
		if _, err := sqlDB.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
}

func names(t *testing.T, sqlDB *sql.DB, query string) map[int64]string {
	t.Helper()
	rows, err := sqlDB.Query(query)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[int64]string{}
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			t.Fatal(err)
		}
		out[id] = name
	}
	return out
}

func assertRenamed(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	cats := names(t, sqlDB, `SELECT id, name FROM categories WHERE id >= 10`)
	for id, want := range map[int64]string{10: "groceries (1)", 11: "GROCERIES (2)", 12: "Groceries"} {
		if cats[id] != want {
			t.Errorf("category %d = %q, want %q", id, cats[id], want)
		}
	}
	if first := names(t, sqlDB, `SELECT id, name FROM categories WHERE id = 1`)[1]; first != "Groceries" {
		t.Errorf("lowest id must keep its name, got %q", first)
	}
	// "Food (1)" already belongs to id 3, so the repeat of "Food" skips to (2).
	tags := names(t, sqlDB, `SELECT id, name FROM tags`)
	for id, want := range map[int64]string{1: "Food", 2: "food (2)", 3: "Food (1)"} {
		if tags[id] != want {
			t.Errorf("tag %d = %q, want %q", id, tags[id], want)
		}
	}
}

func TestUpgradeDedupesNamesAndConformsSchema(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "database.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	createLaravelShape(t, sqlDB)
	addRepeatedNames(t, sqlDB)

	if err := Upgrade(ctx, sqlDB, ""); err != nil {
		t.Fatal(err)
	}
	assertRenamed(t, sqlDB)
	assertConformed(t, sqlDB)
	assertMinorUnits(t, sqlDB)

	// A repeated name is now rejected, whatever its case.
	if _, err := sqlDB.Exec(`INSERT INTO categories (name, type) VALUES ('GROCERIES', 'expense')`); err == nil {
		t.Error("expected unique violation for a repeated category name")
	}
	// ids stay monotonic across the rebuild: the fixture's highest id was 5.
	res, err := sqlDB.Exec(`INSERT INTO transactions (type, account_id, amount, date) VALUES ('expense', 1, 100, '2026-02-01')`)
	if err != nil {
		t.Fatal(err)
	}
	if id, _ := res.LastInsertId(); id <= 5 {
		t.Errorf("new transaction id = %d, want > 5", id)
	}
}

func TestCopyDedupesNames(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	src, err := db.Open(filepath.Join(dir, "laravel.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = src.Close() })
	createLaravelShape(t, src)
	addRepeatedNames(t, src)

	dest, err := db.Open(filepath.Join(dir, "go.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dest.Close() })
	if err := migrate.Up(ctx, dest); err != nil {
		t.Fatal(err)
	}
	if err := Copy(ctx, dest, src); err != nil {
		t.Fatal(err)
	}
	assertRenamed(t, dest)

	// Idempotent: a second copy neither duplicates nor renames again.
	if err := Copy(ctx, dest, src); err != nil {
		t.Fatal(err)
	}
	assertRenamed(t, dest)
}

// assertConformed checks every table is STRICT, retired columns are gone and no
// foreign key is left dangling by the rebuild.
func assertConformed(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	rows, err := sqlDB.Query(`SELECT name FROM pragma_table_list WHERE schema = 'main' AND type = 'table' AND strict = 0 AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var name string
		_ = rows.Scan(&name)
		t.Errorf("table %s is not STRICT", name)
	}
	_ = rows.Close()

	var n int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('transactions') WHERE name = 'exchange_rate'`).Scan(&n); err != nil || n != 0 {
		t.Errorf("transactions.exchange_rate should be gone (%d, %v)", n, err)
	}
	fk, err := sqlDB.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	defer fk.Close()
	if fk.Next() {
		t.Error("foreign_key_check reported violations")
	}
	var idx int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name IN ('categories_type_name_unique', 'currencies_single_base_unique', 'transactions_account_dedup_unique')`).Scan(&idx); err != nil || idx != 3 {
		t.Errorf("declared indexes present = %d, %v; want 3", idx, err)
	}
}
