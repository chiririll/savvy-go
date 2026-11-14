package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"savvy-go/internal/auth"
	"savvy-go/internal/signing"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
	"savvy-go/internal/transfers"
)

// --- restoring -------------------------------------------------------------

// prepareSpaceFrom unpacks a space backup archive and has the store validate
// it. maxBytes bounds the unpacked size.
func (s Service) prepareSpaceFrom(
	ctx context.Context,
	path string,
	maxBytes int64,
) (*store.PreparedSpace, *Manifest, error) {
	arc, err := openArchive(path, s.tmpDir(), maxBytes, s.Keys)
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(arc.Dir)
	if arc.Manifest.Kind != kindSpace {
		return nil, nil, ErrNotSpaceBackup
	}
	p, err := s.Store.PrepareSpace(ctx, filepath.Join(arc.Dir, "space.sqlite"))
	return p, &arc.Manifest, err
}

// RestoreSpace replaces a space with one of its backups. The backup must be
// of this very space (P36) and fit its quota (P39).
func (s Service) RestoreSpace(ctx context.Context, sp spaces.Space, name string, quota int64) error {
	if _, err := s.SpaceBackup(sp.ID, name); err != nil {
		return err
	}
	p, _, err := s.prepareSpaceFrom(ctx, s.SpacePath(sp.ID, name), quota)
	if err != nil {
		return err
	}
	if p.UUID != sp.UUID {
		s.Store.Discard(p.Artifact)
		return ErrOtherSpace
	}
	return s.Replace(ctx, sp, p, quota)
}

// Replace puts a prepared database in the place of a space's own, within quota.
func (s Service) Replace(ctx context.Context, sp spaces.Space, p *store.PreparedSpace, quota int64) error {
	if quota > 0 && p.Size > quota {
		s.Store.Discard(p.Artifact)
		return ErrTooLarge
	}
	// The review mark goes in before the file goes live: if the process dies
	// before the review merge, startup reviews again instead of merging
	// automatically (P24).
	transfers.MarkTransferReview(p)
	if err := s.Store.ReplaceSpace(ctx, sp.ID, p); err != nil {
		return err
	}
	if s.Transfers != nil {
		return s.Transfers.AfterRestore(ctx, sp.ID)
	}
	return nil
}

// ImportSpace creates a new space from a backup file at path, owned by
// owner. The space keeps the backup's uuid unless that uuid is taken on this
// server (a copy next to its original), so spaces moved together between
// servers still recognise each other.
func (s Service) ImportSpace(
	ctx context.Context,
	spaces spaces.Service,
	path, name string,
	owner *auth.User,
	quota int64,
) (*spaces.Space, error) {
	p, m, err := s.prepareSpaceFrom(ctx, path, quota)
	if err != nil {
		return nil, err
	}
	if quota > 0 && p.Size > quota {
		s.Store.Discard(p.Artifact)
		return nil, ErrTooLarge
	}
	if name == "" && m != nil {
		name = m.SpaceName
	}
	if name == "" {
		name = "Imported"
	}
	return spaces.CreateFrom(ctx, name, owner, p)
}

// RestoreServer replaces the server and every space with a server backup.
func (s Service) RestoreServer(ctx context.Context, name string) error {
	if _, err := s.ServerBackup(name); err != nil {
		return err
	}
	arc, err := openArchive(s.ServerPath(name), s.tmpDir(), 0, s.Keys)
	if err != nil {
		return err
	}
	defer os.RemoveAll(arc.Dir)
	if arc.Manifest.Kind != kindServer {
		return ErrNotServerBackup
	}
	var seed []byte
	if raw, err := os.ReadFile(filepath.Join(arc.Dir, "keys", "server.ed25519")); err == nil {
		key, err := signing.Parse(raw)
		if err != nil {
			return err
		}
		seed = key.Seed()
	}
	p, err := s.Store.PrepareServer(ctx, arc.Dir)
	if err != nil {
		return err
	}
	if err := s.Store.ReplaceServer(ctx, p); err != nil {
		return err
	}
	if seed != nil {
		if err := signing.Write(s.DataDir, seed); err != nil {
			return fmt.Errorf("restore signing key: %w", err)
		}
		key, err := signing.Load(s.DataDir, true)
		if err != nil {
			return err
		}
		s.Keys.Holder.Set(key)
	}
	return nil
}
