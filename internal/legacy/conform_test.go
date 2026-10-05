package legacy

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"savvy-go/internal/db"
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

// TestConformRebuildsOldTables degrades a table to the pre-STRICT shape, with
// an extra retired column, and expects conform to restore the current
// definition without losing rows or the id sequence.
func TestConformRebuildsOldTables(t *testing.T) {
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "database.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	ctx := context.Background()
	if err := upBoth(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	exec := func(q string) {
		t.Helper()
		if _, err := sqlDB.ExecContext(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	exec(`PRAGMA foreign_keys=OFF`)
	exec(`CREATE TABLE tags_old (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE, created_at TEXT, updated_at TEXT, retired TEXT)`)
	exec(`DROP TABLE tags`)
	exec(`ALTER TABLE tags_old RENAME TO tags`)
	exec(`PRAGMA foreign_keys=ON`)
	exec(`INSERT INTO tags (id, name, retired) VALUES (1, 'a', 'x'), (2, 'b', 'y'), (3, 'c', 'z')`)
	exec(`DELETE FROM tags WHERE id = 3`) // the sequence stays at 3

	if err := conform(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	if err := conform(ctx, sqlDB); err != nil { // idempotent
		t.Fatal(err)
	}

	var strict, retired int
	if err := sqlDB.QueryRowContext(ctx, `SELECT strict FROM pragma_table_list WHERE name = 'tags' AND schema = 'main'`).Scan(&strict); err != nil || strict != 1 {
		t.Fatalf("tags strict = %d, %v", strict, err)
	}
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('tags') WHERE name = 'retired'`).Scan(&retired); err != nil || retired != 0 {
		t.Fatalf("retired column kept (%d, %v)", retired, err)
	}
	var n int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM tags WHERE name IN ('a', 'b')`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("rows kept = %d, %v", n, err)
	}
	res, err := sqlDB.ExecContext(ctx, `INSERT INTO tags (name) VALUES ('d')`)
	if err != nil {
		t.Fatal(err)
	}
	if id, _ := res.LastInsertId(); id != 4 {
		t.Errorf("next id = %d, want 4 (sequence carried over)", id)
	}
	var enforced int
	if err := sqlDB.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&enforced); err != nil || enforced != 1 {
		t.Errorf("foreign_keys left %d, %v", enforced, err)
	}
}

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
