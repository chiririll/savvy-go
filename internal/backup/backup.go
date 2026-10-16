package backup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/platform/version"
	"savvy-go/internal/signing"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
	"savvy-go/internal/transfers"
)

// Backups of the whole server live in <Dir>/server, backups of a space in
// <Dir>/spaces/<id>. A server backup holds every database and the signing
// key, so only server admins see them; a space backup holds one space and is
// managed by that space's admins.

const (
	archiveExt = ".zip"
)

// Entry is a backup file as listed.
type Entry struct {
	Filename   string
	Size       int64
	Kind       string // kindServer, kindSpace, or "" for a raw database file
	Note       *string
	AppVersion *string
	SpaceUUID  string
	SpaceName  string
	Signature  signing.Signature
	Valid      bool
	CreatedAt  *time.Time
}

// Service creates, lists and restores backups.
type Service struct {
	Store store.Store
	Keys  signing.KeyRing
	Dir   string
	// DataDir is where the signing key file lives (restored with the server).
	DataDir string
	// KeepPerSpace is how many backups a space keeps; older ones are removed.
	// Zero keeps all.
	KeepPerSpace int
	// Transfers reviews a restored space's transfers with its linked spaces.
	Transfers *transfers.Service
}

var (
	// ErrBackupNotFound is returned for a name that is not a backup here.
	ErrBackupNotFound = errors.New("backup not found")
	// ErrOtherSpace is returned when a space backup belongs to another space.
	ErrOtherSpace = errors.New("this backup belongs to another space; import it as a new space instead")
	// ErrNotServerBackup / ErrNotSpaceBackup reject the wrong kind of backup.
	ErrNotServerBackup = errors.New("this is not a server backup")
	ErrNotSpaceBackup  = errors.New("this is not a space backup")
	// ErrTooLarge is returned when a backup would exceed the space quota.
	ErrTooLarge = errors.New("the backup is larger than the space may hold")
)

func (s Service) serverDir() string { return filepath.Join(s.Dir, kindServer) }
func (s Service) spaceDir(id int64) string {
	return filepath.Join(s.Dir, "spaces", strconv.FormatInt(id, 10))
}
func (s Service) tmpDir() string { return filepath.Join(s.Dir, ".tmp") }
func stamp(t time.Time) string   { return t.UTC().Format("20060102-150405") }
func validName(name string) bool {
	return name != "" && filepath.Base(name) == name && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, archiveExt)
}

// --- listing ---------------------------------------------------------------

// ServerBackups lists the server backups, newest first.
func (s Service) ServerBackups() ([]Entry, error) { return s.list(s.serverDir()) }

// SpaceBackups lists a space's backups, newest first.
func (s Service) SpaceBackups(spaceID int64) ([]Entry, error) { return s.list(s.spaceDir(spaceID)) }

func (s Service) list(dir string) ([]Entry, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []Entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []Entry{}
	for _, e := range entries {
		if e.IsDir() || !validName(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, s.describe(dir, info))
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if !a.CreatedAt.Equal(*b.CreatedAt) {
			return a.CreatedAt.After(*b.CreatedAt)
		}
		return a.Filename > b.Filename
	})
	return out, nil
}

func (s Service) describe(dir string, info os.FileInfo) Entry {
	mod := info.ModTime().UTC()
	b := Entry{Filename: info.Name(), Size: info.Size(), CreatedAt: &mod, Signature: signing.Unsigned}
	m, sig, err := readManifest(filepath.Join(dir, info.Name()), s.Keys)
	if err != nil {
		return b
	}
	b.Valid, b.Kind, b.Signature, b.SpaceUUID, b.SpaceName = true, m.Kind, sig, m.SpaceUUID, m.SpaceName
	if m.Note != "" {
		note := m.Note
		b.Note = &note
	}
	if m.AppVersion != "" {
		v := m.AppVersion
		b.AppVersion = &v
	}
	if t, err := time.Parse(time.RFC3339, m.CreatedAt); err == nil {
		b.CreatedAt = &t
	}
	return b
}

// ServerBackup / SpaceBackup find one backup by name.
func (s Service) ServerBackup(name string) (*Entry, error) { return s.find(s.serverDir(), name) }
func (s Service) SpaceBackup(spaceID int64, name string) (*Entry, error) {
	return s.find(s.spaceDir(spaceID), name)
}

func (s Service) find(dir, name string) (*Entry, error) {
	if !validName(name) {
		return nil, ErrBackupNotFound
	}
	info, err := os.Stat(filepath.Join(dir, name))
	if err != nil || info.IsDir() {
		return nil, ErrBackupNotFound
	}
	b := s.describe(dir, info)
	return &b, nil
}

// ServerPath / SpacePath are the files to download.
func (s Service) ServerPath(name string) string { return filepath.Join(s.serverDir(), name) }
func (s Service) SpacePath(spaceID int64, name string) string {
	return filepath.Join(s.spaceDir(spaceID), name)
}

// DeleteServer / DeleteSpace remove a backup.
func (s Service) DeleteServer(name string) error { return s.remove(s.serverDir(), name) }
func (s Service) DeleteSpace(spaceID int64, name string) error {
	return s.remove(s.spaceDir(spaceID), name)
}

func (s Service) remove(dir, name string) error {
	if !validName(name) {
		return ErrBackupNotFound
	}
	err := os.Remove(filepath.Join(dir, name))
	if errors.Is(err, os.ErrNotExist) {
		return ErrBackupNotFound
	}
	return err
}

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
	b, err := s.write(s.spaceDir(sp.ID), Manifest{Kind: kindSpace, Note: deref(note), SpaceUUID: sp.UUID, SpaceName: sp.Name},
		map[string]string{"space.sqlite": db})
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
	m.Format, m.Store, m.AppVersion, m.CreatedAt = 1, "sqlite", version.Value, now.Format(time.RFC3339)
	name := stamp(now) + archiveExt
	for i := 2; fileExists(filepath.Join(dir, name)); i++ {
		name = stamp(now) + "-" + strconv.Itoa(i) + archiveExt
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
	name := stamp(time.Now()) + "-upload" + archiveExt
	for i := 2; fileExists(filepath.Join(dir, name)); i++ {
		name = stamp(time.Now()) + "-upload-" + strconv.Itoa(i) + archiveExt
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

// --- restoring -------------------------------------------------------------

// prepareSpaceFrom unpacks a space backup archive and has the store validate
// it. maxBytes bounds the unpacked size.
func (s Service) prepareSpaceFrom(ctx context.Context, path string, maxBytes int64) (*store.PreparedSpace, *Manifest, error) {
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
func (s Service) ImportSpace(ctx context.Context, spaces spaces.Service, path, name string, owner *auth.User, quota int64) (*spaces.Space, error) {
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

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
