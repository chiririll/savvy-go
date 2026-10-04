// Package sqlite implements store.Store with SQLite files: server.sqlite for
// the server database and spaces/<id>.sqlite for each space. Nothing outside
// this package knows about the files.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	sqlite3 "modernc.org/sqlite"

	"savvy-go/internal/db"
	"savvy-go/internal/migrate"
	"savvy-go/internal/store"
)

const (
	serverFile = "server.sqlite"
	spacesDir  = "spaces"
	spaceExt   = ".sqlite"
	// sqliteFull is the primary result code SQLITE_FULL; extended codes keep
	// it in the low byte.
	sqliteFull = 13
)

// Migrate brings a freshly opened database up to the current schema.
type Migrate func(context.Context, *sql.DB) error

type Options struct {
	// Dir holds server.sqlite and the spaces directory.
	Dir           string
	MigrateServer Migrate
	MigrateSpace  Migrate
	// Quota is the size limit of a space in bytes; 0 means unlimited.
	Quota func(id int64) int64
	// BetweenCommits, when set, runs after each commit of InSpaces except the
	// last. An error stops the remaining commits, simulating a crash between
	// them; tests use it to check that a later merge reconciles the spaces.
	BetweenCommits func(committed int) error
}

// Store is the SQLite store.Store.
type Store struct {
	opts   Options
	server *sql.DB

	mu          sync.Mutex
	spaces      map[int64]*space
	unavailable map[int64]string
	ready       atomic.Bool
}

// space is one space database. gate is held shared for every statement and
// exclusively while the file is swapped or removed; write serialises
// InSpaces calls and exclusive operations, always taken in id order.
type space struct {
	id    int64
	gate  sync.RWMutex
	write sync.Mutex
	db    *sql.DB
	gone  bool
}

var _ store.Store = (*Store)(nil)

// Open opens and migrates the server database and every space file. A space
// that fails to open or migrate is reported unavailable instead of failing
// the whole store.
func Open(ctx context.Context, opts Options) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(opts.Dir, spacesDir), 0o775); err != nil {
		return nil, fmt.Errorf("create spaces dir: %w", err)
	}
	server, err := db.Open(filepath.Join(opts.Dir, serverFile))
	if err != nil {
		return nil, err
	}
	if opts.MigrateServer != nil {
		if err := opts.MigrateServer(ctx, server); err != nil {
			_ = server.Close()
			return nil, fmt.Errorf("migrate server: %w", err)
		}
	}
	s := &Store{opts: opts, server: server, spaces: map[int64]*space{}, unavailable: map[int64]string{}}
	ids, err := s.spaceFiles()
	if err != nil {
		_ = server.Close()
		return nil, err
	}
	for _, id := range ids {
		sdb, err := s.openSpace(ctx, id)
		if err != nil {
			slog.Error("space unavailable", "space", id, "err", err)
			s.unavailable[id] = err.Error()
			continue
		}
		s.spaces[id] = &space{id: id, db: sdb}
	}
	s.ready.Store(true)
	return s, nil
}

func (s *Store) spacePath(id int64) string {
	return filepath.Join(s.opts.Dir, spacesDir, strconv.FormatInt(id, 10)+spaceExt)
}

// spaceFiles lists the ids of the space files on disk.
func (s *Store) spaceFiles() ([]int64, error) {
	entries, err := os.ReadDir(filepath.Join(s.opts.Dir, spacesDir))
	if err != nil {
		return nil, err
	}
	var ids []int64
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), spaceExt)
		if e.IsDir() || !ok {
			continue
		}
		id, err := strconv.ParseInt(name, 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids, nil
}

// openSpace opens and migrates a space file. The quota goes into the DSN as
// max_page_count so every pooled connection enforces it.
func (s *Store) openSpace(ctx context.Context, id int64) (*sql.DB, error) {
	path := s.spacePath(id)
	sdb, err := db.Open(path)
	if err != nil {
		return nil, err
	}
	if s.opts.MigrateSpace != nil {
		if err := s.opts.MigrateSpace(ctx, sdb); err != nil {
			_ = sdb.Close()
			return nil, fmt.Errorf("migrate: %w", err)
		}
	}
	quota := int64(0)
	if s.opts.Quota != nil {
		quota = s.opts.Quota(id)
	}
	if quota <= 0 {
		return sdb, nil
	}
	var pageSize int64
	if err := sdb.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err != nil {
		_ = sdb.Close()
		return nil, err
	}
	_ = sdb.Close()
	return db.Open(path, fmt.Sprintf("max_page_count(%d)", max(quota/pageSize, 1)))
}

func (s *Store) Server() store.DB { return s.server }

func (s *Store) lookup(id int64) (*space, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sp, ok := s.spaces[id]; ok {
		return sp, nil
	}
	if _, ok := s.unavailable[id]; ok {
		return nil, store.ErrUnavailable
	}
	return nil, store.ErrNotFound
}

func (s *Store) Space(_ context.Context, id int64) (store.DB, error) {
	sp, err := s.lookup(id)
	if err != nil {
		return nil, err
	}
	return handle{sp}, nil
}

func (s *Store) CreateSpace(ctx context.Context, id int64) error {
	s.mu.Lock()
	_, exists := s.spaces[id]
	_, broken := s.unavailable[id]
	s.mu.Unlock()
	if exists || broken {
		return fmt.Errorf("space %d already exists", id)
	}
	if _, err := os.Stat(s.spacePath(id)); err == nil {
		return fmt.Errorf("space %d already exists", id)
	}
	sdb, err := s.openSpace(ctx, id)
	if err != nil {
		removeFiles(s.spacePath(id))
		return err
	}
	s.mu.Lock()
	s.spaces[id] = &space{id: id, db: sdb}
	s.mu.Unlock()
	return nil
}

func (s *Store) DeleteSpace(ctx context.Context, id int64) error {
	s.mu.Lock()
	sp, ok := s.spaces[id]
	_, broken := s.unavailable[id]
	s.mu.Unlock()
	if !ok && !broken {
		return store.ErrNotFound
	}
	if ok {
		unlock := s.exclusive(sp)
		defer unlock()
		sp.gone = true
		if err := sp.db.Close(); err != nil {
			return err
		}
	}
	s.mu.Lock()
	delete(s.spaces, id)
	delete(s.unavailable, id)
	s.mu.Unlock()
	removeFiles(s.spacePath(id))
	return nil
}

// exclusive takes the write lock and then the gate of sp, the same order
// InSpaces uses, and returns the release function.
func (s *Store) exclusive(sp *space) func() {
	sp.write.Lock()
	sp.gate.Lock()
	return func() {
		sp.gate.Unlock()
		sp.write.Unlock()
	}
}

func (s *Store) Spaces(context.Context) ([]int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]int64, 0, len(s.spaces)+len(s.unavailable))
	for id := range s.spaces {
		ids = append(ids, id)
	}
	for id := range s.unavailable {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids, nil
}

func (s *Store) SpaceSize(ctx context.Context, id int64) (int64, error) {
	sp, err := s.lookup(id)
	if err != nil {
		return 0, err
	}
	h := handle{sp}
	var pages, size int64
	if err := h.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pages); err != nil {
		return 0, err
	}
	if err := h.QueryRowContext(ctx, "PRAGMA page_size").Scan(&size); err != nil {
		return 0, err
	}
	return pages * size, nil
}

// SetQuota reopens a space so a changed quota takes effect.
func (s *Store) SetQuota(ctx context.Context, id int64) error {
	sp, err := s.lookup(id)
	if err != nil {
		return err
	}
	unlock := s.exclusive(sp)
	defer unlock()
	if err := sp.db.Close(); err != nil {
		return err
	}
	sdb, err := s.openSpace(ctx, id)
	if err != nil {
		sp.gone = true
		s.mu.Lock()
		delete(s.spaces, id)
		s.unavailable[id] = err.Error()
		s.mu.Unlock()
		return err
	}
	sp.db = sdb
	return nil
}

func (s *Store) Status() store.Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := store.Status{Ready: s.ready.Load(), Unavailable: map[int64]string{}}
	for id, why := range s.unavailable {
		st.Unavailable[id] = why
	}
	return st
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var errs []error
	for _, sp := range s.spaces {
		sp.gate.Lock()
		sp.gone = true
		errs = append(errs, sp.db.Close())
		sp.gate.Unlock()
	}
	s.spaces = map[int64]*space{}
	errs = append(errs, s.server.Close())
	return errors.Join(errs...)
}

// InSpaces locks the spaces in id order, opens a transaction on each, runs fn
// and commits them one after another. See store.Store.
func (s *Store) InSpaces(ctx context.Context, ids []int64, fn func(map[int64]store.DB) error) error {
	ids = slices.Clone(ids)
	slices.Sort(ids)
	ids = slices.Compact(ids)

	locked := make([]*space, 0, len(ids))
	defer func() {
		for i := len(locked) - 1; i >= 0; i-- {
			locked[i].gate.RUnlock()
			locked[i].write.Unlock()
		}
	}()
	for _, id := range ids {
		sp, err := s.lookup(id)
		if err != nil {
			return fmt.Errorf("space %d: %w", id, err)
		}
		sp.write.Lock()
		sp.gate.RLock()
		locked = append(locked, sp)
		if sp.gone {
			return fmt.Errorf("space %d: %w", id, store.ErrNotFound)
		}
	}

	txs := make([]*sql.Tx, 0, len(locked))
	rollback := func() {
		for _, tx := range txs {
			_ = tx.Rollback()
		}
	}
	handles := make(map[int64]store.DB, len(locked))
	for _, sp := range locked {
		tx, err := sp.db.BeginTx(ctx, nil)
		if err != nil {
			rollback()
			return err
		}
		txs = append(txs, tx)
		handles[sp.id] = txHandle{tx}
	}
	if err := fn(handles); err != nil {
		rollback()
		return mapErr(err)
	}
	for i, tx := range txs {
		if err := tx.Commit(); err != nil {
			for _, rest := range txs[i+1:] {
				_ = rest.Rollback()
			}
			return mapErr(err)
		}
		if s.opts.BetweenCommits != nil && i < len(txs)-1 {
			if err := s.opts.BetweenCommits(i + 1); err != nil {
				for _, rest := range txs[i+1:] {
					_ = rest.Rollback()
				}
				return err
			}
		}
	}
	return nil
}

func removeFiles(path string) {
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		_ = os.Remove(path + suffix)
	}
}

// quotaError keeps the driver error for the log while matching
// store.ErrQuotaExceeded.
type quotaError struct{ err error }

func (e quotaError) Error() string   { return store.ErrQuotaExceeded.Error() + ": " + e.err.Error() }
func (e quotaError) Unwrap() []error { return []error{store.ErrQuotaExceeded, e.err} }

// mapErr turns SQLITE_FULL (the file reached max_page_count) into
// store.ErrQuotaExceeded.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	var se *sqlite3.Error
	if errors.As(err, &se) && se.Code()&0xff == sqliteFull {
		return quotaError{err}
	}
	return err
}

// OpenApp opens the store of the application in dir with the embedded server
// and space migrations.
func OpenApp(ctx context.Context, dir string) (*Store, error) {
	return Open(ctx, Options{Dir: dir, MigrateServer: migrate.Server.Up, MigrateSpace: migrate.Space.Up})
}
