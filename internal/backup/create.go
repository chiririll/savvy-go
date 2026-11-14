package backup

import (
	"context"
	"os"
	"path/filepath"
	"savvy-go/internal/signing"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
	"strconv"
	"time"
)

// --- creating --------------------------------------------------------------

// CreateServer backs up the server and every space.
func (s Service) CreateServer(ctx context.Context, note *string) (*Entry, error) {
	work, err := s.workDir()
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(work)
	if err := s.Store.ExportServer(ctx, work); err != nil {
		return nil, err
	}
	files := map[string]string{"server.sqlite": filepath.Join(work, "server.sqlite")}
	spaces, _ := os.ReadDir(filepath.Join(work, "spaces"))
	for _, e := range spaces {
		files["spaces/"+e.Name()] = filepath.Join(work, "spaces", e.Name())
	}
	key := s.Keys.Holder.Key()
	seedFile := filepath.Join(work, "server.ed25519")
	if err := signing.Write(filepath.Dir(seedFile), key.Seed()); err != nil {
		return nil, err
	}
	files["keys/server.ed25519"] = signing.Path(filepath.Dir(seedFile))
	return s.write(s.serverDir(), Manifest{Kind: kindServer, Note: deref(note)}, files)
}

// CreateSpace backs up one space; the oldest backups beyond KeepPerSpace go.
func (s Service) CreateSpace(ctx context.Context, sp spaces.Space, note *string) (*Entry, error) {
	work, err := s.workDir()
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(work)
	db := filepath.Join(work, "space.sqlite")
	if err := s.Store.ExportSpace(ctx, sp.ID, db); err != nil {
		return nil, err
	}
	b, err := s.write(
		s.spaceDir(sp.ID),
		Manifest{Kind: kindSpace, Note: deref(note), SpaceUUID: sp.UUID, SpaceName: sp.Name},
		map[string]string{"space.sqlite": db},
	)
	if err != nil {
		return nil, err
	}
	s.prune(sp.ID)
	return b, nil
}

func (s Service) write(dir string, m Manifest, files map[string]string) (*Entry, error) {
	if err := os.MkdirAll(dir, 0o775); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	m.Format, m.Store, m.AppVersion, m.CreatedAt = 1, "sqlite", s.AppVersion, now.Format(time.RFC3339)
	base := baseName(m) + "-" + stamp(now)
	name := base + archiveExt
	for i := 2; fileExists(filepath.Join(dir, name)); i++ {
		name = base + "-" + strconv.Itoa(i) + archiveExt
	}
	if err := writeArchive(filepath.Join(dir, name), m, files, s.Keys.Holder.Key()); err != nil {
		return nil, err
	}
	return s.find(dir, name)
}

func (s Service) prune(spaceID int64) {
	if s.KeepPerSpace <= 0 {
		return
	}
	list, err := s.SpaceBackups(spaceID)
	if err != nil {
		return
	}
	for _, b := range list[min(len(list), s.KeepPerSpace):] {
		_ = s.DeleteSpace(spaceID, b.Filename)
	}
}

func (s Service) workDir() (string, error) {
	if err := os.MkdirAll(s.tmpDir(), 0o700); err != nil {
		return "", err
	}
	return os.MkdirTemp(s.tmpDir(), "work-")
}

// IngestServer / IngestSpace move an uploaded file into the backup directory.
func (s Service) IngestServer(src string) (*Entry, error) {
	return s.ingest(s.serverDir(), src)
}

func (s Service) IngestSpace(spaceID int64, src string) (*Entry, error) {
	b, err := s.ingest(s.spaceDir(spaceID), src)
	if err == nil {
		s.prune(spaceID)
	}
	return b, err
}

func (s Service) ingest(dir, src string) (*Entry, error) {
	if err := os.MkdirAll(dir, 0o775); err != nil {
		return nil, err
	}
	name := "upload-" + stamp(time.Now()) + archiveExt
	for i := 2; fileExists(filepath.Join(dir, name)); i++ {
		name = "upload-" + stamp(time.Now()) + "-" + strconv.Itoa(i) + archiveExt
	}
	if err := os.Rename(src, filepath.Join(dir, name)); err != nil {
		return nil, err
	}
	return s.find(dir, name)
}

// SaveServerFrom keeps a prepared server (a converted Laravel database) as a
// server backup and consumes p. It has no signing key, so restoring it leaves
// this server's key as it is.
func (s Service) SaveServerFrom(p *store.PreparedServer, note string) (*Entry, error) {
	defer s.Store.Discard(p.Artifact)
	files := map[string]string{"server.sqlite": filepath.Join(p.Artifact, "server.sqlite")}
	for _, id := range p.Spaces {
		name := strconv.FormatInt(id, 10) + ".sqlite"
		files["spaces/"+name] = filepath.Join(p.Artifact, "spaces", name)
	}
	return s.write(s.serverDir(), Manifest{Kind: kindServer, Note: note}, files)
}

// SaveSpaceFrom keeps a prepared space database (a converted Laravel
// database) as a backup of sp and consumes p. The database must carry sp's
// uuid, or the backup could not be restored over sp.
func (s Service) SaveSpaceFrom(sp spaces.Space, p *store.PreparedSpace, note string) (*Entry, error) {
	defer s.Store.Discard(p.Artifact)
	b, err := s.write(s.spaceDir(sp.ID), Manifest{Kind: kindSpace, Note: note, SpaceUUID: sp.UUID, SpaceName: sp.Name},
		map[string]string{"space.sqlite": filepath.Join(p.Artifact, "space.sqlite")})
	if err != nil {
		return nil, err
	}
	s.prune(sp.ID)
	return b, nil
}

// deletedDir holds the final backups of deleted spaces, for server admins.
func (s Service) deletedDir() string { return filepath.Join(s.Dir, "deleted") }

// deletedKeep is how long final backups of deleted spaces are kept.
const deletedKeep = 30 * 24 * time.Hour

// FinalBackup backs up a space about to be deleted (P41).
func (s Service) FinalBackup(ctx context.Context, sp spaces.Space) error {
	work, err := s.workDir()
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	dbFile := filepath.Join(work, "space.sqlite")
	if err := s.Store.ExportSpace(ctx, sp.ID, dbFile); err != nil {
		return err
	}
	_, err = s.write(s.deletedDir(), Manifest{Kind: kindSpace, Note: "deleted", SpaceUUID: sp.UUID, SpaceName: sp.Name},
		map[string]string{"space.sqlite": dbFile})
	return err
}

// DeletedBackups lists the final backups of deleted spaces, removing those
// older than deletedKeep.
func (s Service) DeletedBackups() ([]Entry, error) {
	list, err := s.list(s.deletedDir())
	if err != nil {
		return nil, err
	}
	out := list[:0]
	for _, b := range list {
		if b.CreatedAt != nil && time.Since(*b.CreatedAt) > deletedKeep {
			_ = s.remove(s.deletedDir(), b.Filename)
			continue
		}
		out = append(out, b)
	}
	return out, nil
}

// DeletedPath is the file of a final backup.
func (s Service) DeletedPath(name string) (string, error) {
	if _, err := s.find(s.deletedDir(), name); err != nil {
		return "", err
	}
	return filepath.Join(s.deletedDir(), name), nil
}
