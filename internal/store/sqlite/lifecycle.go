package sqlite

import (
	"context"
	"fmt"
	"os"

	"savvy-go/internal/db"
	"savvy-go/internal/migrate"
	"savvy-go/internal/settings"
	"savvy-go/internal/store"
)

func (s *Store) CreateSpace(ctx context.Context, id int64) error {
	s.layout.RLock()
	defer s.layout.RUnlock()
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
	s.layout.RLock()
	defer s.layout.RUnlock()
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

// ApplyQuota reopens a space so a changed quota takes effect.
func (s *Store) ApplyQuota(ctx context.Context, id int64) error {
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

// OpenApp opens the store of the application in dir with the embedded server
// and space migrations.
func OpenApp(ctx context.Context, dir string) (*Store, error) {
	return Open(ctx, Options{
		Dir: dir, MigrateServer: migrate.Server.Up, MigrateSpace: migrate.Space.Up,
		Quota: func(server store.DB, id int64) int64 {
			if q, err := db.Q(server).GetSpaceQuota(ctx, id); err == nil && q.Valid {
				return q.Int64
			}
			return settings.SpaceQuota(ctx, server)
		},
	})
}
