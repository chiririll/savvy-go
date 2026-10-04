package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"savvy-go/internal/store"
)

const testSchema = `CREATE TABLE IF NOT EXISTS notes (id INTEGER PRIMARY KEY, body TEXT NOT NULL)`

func migrateNotes(ctx context.Context, d *sql.DB) error {
	_, err := d.ExecContext(ctx, testSchema)
	return err
}

func openTest(t *testing.T, opts Options) *Store {
	t.Helper()
	if opts.Dir == "" {
		opts.Dir = t.TempDir()
	}
	if opts.MigrateSpace == nil {
		opts.MigrateSpace = migrateNotes
	}
	s, err := Open(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func insertNote(t *testing.T, d store.DB, body string) {
	t.Helper()
	if _, err := d.ExecContext(context.Background(), `INSERT INTO notes (body) VALUES (?)`, body); err != nil {
		t.Fatal(err)
	}
}

func countNotes(t *testing.T, s *Store, id int64) int {
	t.Helper()
	d, err := s.Space(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	var n int
	if err := d.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM notes`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestSpacesAreSeparateFiles(t *testing.T) {
	ctx := context.Background()
	s := openTest(t, Options{})
	for _, id := range []int64{1, 2} {
		if err := s.CreateSpace(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	a, _ := s.Space(ctx, 1)
	insertNote(t, a, "only in 1")
	if countNotes(t, s, 1) != 1 || countNotes(t, s, 2) != 0 {
		t.Fatal("a row written to space 1 is visible in space 2")
	}
	if _, err := s.Space(ctx, 3); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing space: %v", err)
	}
	if err := s.CreateSpace(ctx, 1); err == nil {
		t.Fatal("creating an existing space must fail")
	}
}

func TestReopenFindsSpaceFiles(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s := openTest(t, Options{Dir: dir})
	_ = s.CreateSpace(ctx, 7)
	d, _ := s.Space(ctx, 7)
	insertNote(t, d, "kept")
	_ = s.Close()

	s = openTest(t, Options{Dir: dir})
	ids, _ := s.Spaces(ctx)
	if len(ids) != 1 || ids[0] != 7 || countNotes(t, s, 7) != 1 {
		t.Fatalf("reopened spaces %v", ids)
	}
}

// P15: a space that fails to migrate is unavailable; the others work.
func TestP15FailedMigrationMarksOnlyThatSpaceUnavailable(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s := openTest(t, Options{Dir: dir})
	_ = s.CreateSpace(ctx, 1)
	_ = s.CreateSpace(ctx, 2)
	broken, _ := s.Space(ctx, 2)
	insertNote(t, broken, "break-me")
	_ = s.Close()

	s = openTest(t, Options{Dir: dir, MigrateSpace: failFor})
	st := s.Status()
	if !st.Ready || len(st.Unavailable) != 1 {
		t.Fatalf("status %+v", st)
	}
	if _, ok := st.Unavailable[2]; !ok {
		t.Fatalf("space 2 should be unavailable: %+v", st)
	}
	if _, err := s.Space(ctx, 2); !errors.Is(err, store.ErrUnavailable) {
		t.Fatalf("unavailable space: %v", err)
	}
	if countNotes(t, s, 1) != 0 {
		t.Fatal("space 1 must keep working")
	}
	if err := s.InSpaces(ctx, []int64{1, 2}, func(map[int64]store.DB) error { return nil }); !errors.Is(err, store.ErrUnavailable) {
		t.Fatalf("InSpaces with an unavailable space: %v", err)
	}
}

// failFor fails the migration of the space whose notes table holds a marker.
func failFor(ctx context.Context, d *sql.DB) error {
	var marked int
	_ = d.QueryRowContext(ctx, `SELECT COUNT(*) FROM notes WHERE body = 'break-me'`).Scan(&marked)
	if marked > 0 {
		return errors.New("broken migration")
	}
	return nil
}

// P14: the quota stops writes with ErrQuotaExceeded; other spaces go on.
func TestP14QuotaExceeded(t *testing.T) {
	ctx := context.Background()
	s := openTest(t, Options{Quota: func(id int64) int64 {
		if id == 1 {
			return 64 * 1024
		}
		return 0
	}})
	_ = s.CreateSpace(ctx, 1)
	_ = s.CreateSpace(ctx, 2)
	a, _ := s.Space(ctx, 1)
	big := strings.Repeat("x", 4096)
	var err error
	for i := 0; i < 100 && err == nil; i++ {
		_, err = a.ExecContext(ctx, `INSERT INTO notes (body) VALUES (?)`, big)
	}
	if !errors.Is(err, store.ErrQuotaExceeded) {
		t.Fatalf("want ErrQuotaExceeded, got %v", err)
	}
	b, _ := s.Space(ctx, 2)
	for i := 0; i < 100; i++ {
		insertNote(t, b, big)
	}
	if size, _ := s.SpaceSize(ctx, 1); size > 64*1024 {
		t.Fatalf("space 1 grew to %d bytes", size)
	}
}

// P39: the quota is in the DSN, so a connection opened later enforces it too.
func TestP39QuotaAppliesToEveryConnection(t *testing.T) {
	ctx := context.Background()
	s := openTest(t, Options{Quota: func(int64) int64 { return 40960 }})
	_ = s.CreateSpace(ctx, 1)
	sp, _ := s.lookup(1)
	sp.db.SetMaxOpenConns(2)
	c1, err := sp.db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer c1.Close()
	c2, err := sp.db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer c2.Close()
	for i, c := range []*sql.Conn{c1, c2} {
		var pages int64
		if err := c.QueryRowContext(ctx, `PRAGMA max_page_count`).Scan(&pages); err != nil {
			t.Fatal(err)
		}
		if pages != 10 {
			t.Fatalf("connection %d: max_page_count %d, want 10", i, pages)
		}
	}
}

// P22: a failure inside fn on the second space rolls back both.
func TestP22InSpacesRollsBackAll(t *testing.T) {
	ctx := context.Background()
	s := openTest(t, Options{})
	_ = s.CreateSpace(ctx, 1)
	_ = s.CreateSpace(ctx, 2)
	err := s.InSpaces(ctx, []int64{2, 1}, func(dbs map[int64]store.DB) error {
		insertNote(t, dbs[1], "a")
		insertNote(t, dbs[2], "b")
		return errors.New("second side failed")
	})
	if err == nil {
		t.Fatal("want error")
	}
	if countNotes(t, s, 1) != 0 || countNotes(t, s, 2) != 0 {
		t.Fatal("nothing may be written when fn fails")
	}
	if err := s.InSpaces(ctx, []int64{1, 2}, func(dbs map[int64]store.DB) error {
		insertNote(t, dbs[1], "a")
		insertNote(t, dbs[2], "b")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if countNotes(t, s, 1) != 1 || countNotes(t, s, 2) != 1 {
		t.Fatal("both sides must be committed")
	}
}

// P22/P29: a quota failure on one side writes neither.
func TestP29QuotaOnOneSideWritesNeither(t *testing.T) {
	ctx := context.Background()
	s := openTest(t, Options{Quota: func(id int64) int64 {
		if id == 2 {
			return 40960
		}
		return 0
	}})
	_ = s.CreateSpace(ctx, 1)
	_ = s.CreateSpace(ctx, 2)
	big := strings.Repeat("y", 8192)
	err := s.InSpaces(ctx, []int64{1, 2}, func(dbs map[int64]store.DB) error {
		if _, err := dbs[1].ExecContext(ctx, `INSERT INTO notes (body) VALUES (?)`, big); err != nil {
			return err
		}
		for i := 0; i < 20; i++ {
			if _, err := dbs[2].ExecContext(ctx, `INSERT INTO notes (body) VALUES (?)`, big); err != nil {
				return err
			}
		}
		return nil
	})
	if !errors.Is(err, store.ErrQuotaExceeded) {
		t.Fatalf("want ErrQuotaExceeded, got %v", err)
	}
	if countNotes(t, s, 1) != 0 || countNotes(t, s, 2) != 0 {
		t.Fatal("a quota failure must leave both spaces unchanged")
	}
}

// P22: a crash between commits leaves the first space committed only; the
// hook is how the merge tests reproduce it.
func TestP22CrashBetweenCommits(t *testing.T) {
	ctx := context.Background()
	s := openTest(t, Options{BetweenCommits: func(int) error { return errors.New("crash") }})
	_ = s.CreateSpace(ctx, 1)
	_ = s.CreateSpace(ctx, 2)
	err := s.InSpaces(ctx, []int64{1, 2}, func(dbs map[int64]store.DB) error {
		insertNote(t, dbs[1], "a")
		insertNote(t, dbs[2], "b")
		return nil
	})
	if err == nil {
		t.Fatal("want the simulated crash")
	}
	if countNotes(t, s, 1) != 1 || countNotes(t, s, 2) != 0 {
		t.Fatal("want space 1 committed and space 2 not")
	}
}

// P22/P54: concurrent A→B and B→A operations do not deadlock.
func TestP22OppositeOrderNoDeadlock(t *testing.T) {
	ctx := context.Background()
	s := openTest(t, Options{})
	_ = s.CreateSpace(ctx, 1)
	_ = s.CreateSpace(ctx, 2)
	done := make(chan struct{})
	go func() {
		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			for _, ids := range [][]int64{{1, 2}, {2, 1}} {
				wg.Add(1)
				go func(ids []int64) {
					defer wg.Done()
					_ = s.InSpaces(ctx, ids, func(dbs map[int64]store.DB) error {
						for _, id := range ids {
							if _, err := dbs[id].ExecContext(ctx, `INSERT INTO notes (body) VALUES ('x')`); err != nil {
								return err
							}
						}
						return nil
					})
				}(ids)
			}
		}
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("deadlock")
	}
	if countNotes(t, s, 1) != 40 || countNotes(t, s, 2) != 40 {
		t.Fatalf("lost writes: %d %d", countNotes(t, s, 1), countNotes(t, s, 2))
	}
}

// P17: statements racing a delete either finish or fail cleanly.
func TestP17DeleteWhileQuerying(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s := openTest(t, Options{Dir: dir})
	_ = s.CreateSpace(ctx, 1)
	d, _ := s.Space(ctx, 1)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				if _, err := d.ExecContext(ctx, `INSERT INTO notes (body) VALUES ('x')`); err != nil {
					if !errors.Is(err, store.ErrNotFound) {
						t.Errorf("unexpected error %v", err)
					}
					return
				}
			}
		}()
	}
	time.Sleep(5 * time.Millisecond)
	if err := s.DeleteSpace(ctx, 1); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	if _, err := os.Stat(filepath.Join(dir, spacesDir, "1.sqlite")); !os.IsNotExist(err) {
		t.Fatal("space file must be removed")
	}
	if _, err := d.ExecContext(ctx, `SELECT 1`); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("handle of a deleted space: %v", err)
	}
}

func TestTxJoinsInSpacesTransaction(t *testing.T) {
	ctx := context.Background()
	s := openTest(t, Options{})
	_ = s.CreateSpace(ctx, 1)
	err := s.InSpaces(ctx, []int64{1}, func(dbs map[int64]store.DB) error {
		return store.Tx(ctx, dbs[1], func(tx store.DB) error {
			insertNote(t, tx, "nested")
			return errors.New("abort")
		})
	})
	if err == nil || countNotes(t, s, 1) != 0 {
		t.Fatal("a nested Tx must join the outer transaction")
	}
	d, _ := s.Space(ctx, 1)
	if err := store.Tx(ctx, d, func(tx store.DB) error { insertNote(t, tx, "ok"); return nil }); err != nil {
		t.Fatal(err)
	}
	if countNotes(t, s, 1) != 1 {
		t.Fatal("Tx on a space handle must commit")
	}
}
