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

	"savvy-go/internal/db"
	"savvy-go/internal/store"
)

const (
	serverFile = "server.sqlite"
	spacesDir  = "spaces"
	spaceExt   = ".sqlite"
	// sqliteFull is the primary result code SQLITE_FULL; extended codes keep
	// it in the low byte.
	sqliteFull = 13
	// sqliteConstraint is SQLITE_CONSTRAINT (CHECK, UNIQUE, NOT NULL, foreign key).
	sqliteConstraint = 19
)

// Migrate brings a freshly opened database up to the current schema.
type Migrate func(context.Context, *sql.DB) error

type Options struct {
	// Dir holds server.sqlite and the spaces directory.
	Dir string
	// MigrateServer and MigrateSpace may be nil: the files are then opened as
	// they are, without touching the schema.
	MigrateServer Migrate
	MigrateSpace  Migrate
	// Quota is the size limit of a space in bytes; 0 means unlimited. It may
	// read the server database it is given.
	Quota func(server store.DB, id int64) int64
	// BetweenCommits, when set, runs after each commit of InSpaces except the
	// last. An error stops the remaining commits, simulating a crash between
	// them; tests use it to check that a later merge reconciles the spaces.
	BetweenCommits func(committed int) error
}

// Store is the SQLite store.Store.
type Store struct {
	opts Options

	// serverGate guards server: shared for statements, exclusive while a
	// server restore swaps the file.
	serverGate sync.RWMutex
	server     *sql.DB

	// layout is held shared while a space is created, removed or imported and
	// exclusively while the whole store is exported or replaced.
	layout sync.RWMutex

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
	s := &Store{opts: opts, spaces: map[int64]*space{}, unavailable: map[int64]string{}}
	if err := s.openAll(ctx); err != nil {
		return nil, err
	}
	s.ready.Store(true)
	return s, nil
}

// openAll opens and migrates the server file and every space file.
func (s *Store) openAll(ctx context.Context) error {
	server, err := db.Open(filepath.Join(s.opts.Dir, serverFile))
	if err != nil {
		return err
	}
	if s.opts.MigrateServer != nil {
		if err := s.opts.MigrateServer(ctx, server); err != nil {
			_ = server.Close()
			return fmt.Errorf("migrate server: %w", err)
		}
	}
	ids, err := s.spaceFiles()
	if err != nil {
		_ = server.Close()
		return err
	}
	s.server = server
	s.mu.Lock()
	defer s.mu.Unlock()
	s.spaces = map[int64]*space{}
	s.unavailable = map[int64]string{}
	for _, id := range ids {
		sdb, err := s.openSpace(ctx, id)
		if err != nil {
			slog.Error("space unavailable", "space", id, "err", err)
			s.unavailable[id] = err.Error()
			continue
		}
		s.spaces[id] = &space{id: id, db: sdb}
	}
	return nil
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
		// s.server without the gate: openSpace runs while opening the store,
		// under ReplaceServer (which holds the gate) or under a lock that
		// excludes ReplaceServer.
		quota = s.opts.Quota(s.server, id)
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

// Server returns a handle that stays valid across a server restore.
func (s *Store) Server() store.DB { return serverHandle{s} }

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
	s.serverGate.Lock()
	errs = append(errs, s.server.Close())
	s.serverGate.Unlock()
	return errors.Join(errs...)
}
