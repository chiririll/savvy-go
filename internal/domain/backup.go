package domain

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/signing"
	"savvy-go/internal/store"
	"savvy-go/internal/version"
)

// Backups of the whole server live in <Dir>/server, backups of a space in
// <Dir>/spaces/<id>. A server backup holds every database and the signing
// key, so only server admins see them; a space backup holds one space and is
// managed by that space's admins.

const (
	archiveExt = ".zip"
	// rawExt is an uploaded single database file: the single-file layout or
	// a Laravel-era backup. Such files carry no manifest and no signature.
	rawExt = ".sqlite"
)

// Backup is a backup file as listed.
type Backup struct {
	Filename   string
	Size       int64
	Kind       string // kindServer, kindSpace, or "" for a raw database file
	Note       *string
	AppVersion *string
	SpaceUUID  string
	SpaceName  string
	Signature  Signature
	Valid      bool
	CreatedAt  *time.Time
}

// KeyRing checks signatures: this server's key and the keys it trusts.
type KeyRing struct {
	Holder  *signing.Holder
	Trusted func(kid string) ed25519.PublicKey
}

func (k KeyRing) check(kid string, msg, sig []byte) Signature {
	raw, err := decodeSig(sig)
	if err != nil || k.Holder == nil {
		return Unsigned
	}
	if own := k.Holder.Key(); own != nil && own.KID() == kid && signing.Verify(own.Public(), msg, raw) {
		return SignedHere
	}
	if k.Trusted != nil {
		if pub := k.Trusted(kid); pub != nil && signing.Verify(pub, msg, raw) {
			return SignedTrusted
		}
	}
	return Unsigned
}

func decodeSig(b []byte) ([]byte, error) {
	if len(b) == 0 {
		return nil, errors.New("no signature")
	}
	return base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
}

// Backups creates, lists and restores backups.
type Backups struct {
	Store store.Store
	Keys  KeyRing
	Dir   string
	// DataDir is where the signing key file lives (restored with the server).
	DataDir string
	// KeepPerSpace is how many backups a space keeps; older ones are removed.
	// Zero keeps all.
	KeepPerSpace int
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

func (s Backups) serverDir() string { return filepath.Join(s.Dir, kindServer) }
func (s Backups) spaceDir(id int64) string {
	return filepath.Join(s.Dir, "spaces", strconv.FormatInt(id, 10))
}
func (s Backups) tmpDir() string { return filepath.Join(s.Dir, ".tmp") }
func stamp(t time.Time) string   { return t.UTC().Format("20060102-150405") }
func validName(name string) bool {
	return name != "" && filepath.Base(name) == name && !strings.HasPrefix(name, ".") &&
		(strings.HasSuffix(name, archiveExt) || strings.HasSuffix(name, rawExt))
}

// --- listing ---------------------------------------------------------------

// ServerBackups lists the server backups, newest first.
func (s Backups) ServerBackups() ([]Backup, error) { return s.list(s.serverDir()) }

// SpaceBackups lists a space's backups, newest first.
func (s Backups) SpaceBackups(spaceID int64) ([]Backup, error) { return s.list(s.spaceDir(spaceID)) }

func (s Backups) list(dir string) ([]Backup, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []Backup{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []Backup{}
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

func (s Backups) describe(dir string, info os.FileInfo) Backup {
	mod := info.ModTime().UTC()
	b := Backup{Filename: info.Name(), Size: info.Size(), CreatedAt: &mod, Signature: Unsigned}
	if strings.HasSuffix(info.Name(), rawExt) {
		b.Valid = true
		return b
	}
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
func (s Backups) ServerBackup(name string) (*Backup, error) { return s.find(s.serverDir(), name) }
func (s Backups) SpaceBackup(spaceID int64, name string) (*Backup, error) {
	return s.find(s.spaceDir(spaceID), name)
}

func (s Backups) find(dir, name string) (*Backup, error) {
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
func (s Backups) ServerPath(name string) string { return filepath.Join(s.serverDir(), name) }
func (s Backups) SpacePath(spaceID int64, name string) string {
	return filepath.Join(s.spaceDir(spaceID), name)
}

// DeleteServer / DeleteSpace remove a backup.
func (s Backups) DeleteServer(name string) error { return s.remove(s.serverDir(), name) }
func (s Backups) DeleteSpace(spaceID int64, name string) error {
	return s.remove(s.spaceDir(spaceID), name)
}

func (s Backups) remove(dir, name string) error {
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
func (s Backups) CreateServer(ctx context.Context, note *string) (*Backup, error) {
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
func (s Backups) CreateSpace(ctx context.Context, sp Space, note *string) (*Backup, error) {
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

func (s Backups) write(dir string, m Manifest, files map[string]string) (*Backup, error) {
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

func (s Backups) prune(spaceID int64) {
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

func (s Backups) workDir() (string, error) {
	if err := os.MkdirAll(s.tmpDir(), 0o700); err != nil {
		return "", err
	}
	return os.MkdirTemp(s.tmpDir(), "work-")
}

// IngestServer / IngestSpace move an uploaded file into the backup directory.
func (s Backups) IngestServer(src, original string) (*Backup, error) {
	return s.ingest(s.serverDir(), src, original)
}

func (s Backups) IngestSpace(spaceID int64, src, original string) (*Backup, error) {
	b, err := s.ingest(s.spaceDir(spaceID), src, original)
	if err == nil {
		s.prune(spaceID)
	}
	return b, err
}

func (s Backups) ingest(dir, src, original string) (*Backup, error) {
	ext := archiveExt
	if strings.HasSuffix(strings.ToLower(original), rawExt) {
		ext = rawExt
	}
	if err := os.MkdirAll(dir, 0o775); err != nil {
		return nil, err
	}
	name := stamp(time.Now()) + "-upload" + ext
	for i := 2; fileExists(filepath.Join(dir, name)); i++ {
		name = stamp(time.Now()) + "-upload-" + strconv.Itoa(i) + ext
	}
	if err := os.Rename(src, filepath.Join(dir, name)); err != nil {
		return nil, err
	}
	return s.find(dir, name)
}

// deletedDir holds the final backups of deleted spaces, for server admins.
func (s Backups) deletedDir() string { return filepath.Join(s.Dir, "deleted") }

// deletedKeep is how long final backups of deleted spaces are kept.
const deletedKeep = 30 * 24 * time.Hour

// FinalBackup backs up a space about to be deleted (P41).
func (s Backups) FinalBackup(ctx context.Context, sp Space) (*Backup, error) {
	work, err := s.workDir()
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(work)
	dbFile := filepath.Join(work, "space.sqlite")
	if err := s.Store.ExportSpace(ctx, sp.ID, dbFile); err != nil {
		return nil, err
	}
	return s.write(s.deletedDir(), Manifest{Kind: kindSpace, Note: "deleted", SpaceUUID: sp.UUID, SpaceName: sp.Name},
		map[string]string{"space.sqlite": dbFile})
}

// DeletedBackups lists the final backups of deleted spaces, removing those
// older than deletedKeep.
func (s Backups) DeletedBackups() ([]Backup, error) {
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
func (s Backups) DeletedPath(name string) (string, error) {
	if _, err := s.find(s.deletedDir(), name); err != nil {
		return "", err
	}
	return filepath.Join(s.deletedDir(), name), nil
}

// --- restoring -------------------------------------------------------------

// prepareSpaceFrom unpacks a space backup (or takes a raw database file) and
// has the store validate it. maxBytes bounds the unpacked size.
func (s Backups) prepareSpaceFrom(ctx context.Context, path string, maxBytes int64) (*store.PreparedSpace, *Manifest, error) {
	if strings.HasSuffix(path, rawExt) {
		if maxBytes > 0 && fileSizeOf(path) > maxBytes {
			return nil, nil, ErrTooLarge
		}
		p, err := s.Store.PrepareSpace(ctx, path)
		return p, nil, err
	}
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
func (s Backups) RestoreSpace(ctx context.Context, sp Space, name string, quota int64) error {
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
	if quota > 0 && p.Size > quota {
		s.Store.Discard(p.Artifact)
		return ErrTooLarge
	}
	return s.Store.ReplaceSpace(ctx, sp.ID, p)
}

// ImportSpace creates a new space from a backup file at path, owned by
// owner. The space keeps the backup's uuid unless that uuid is taken on this
// server (a copy next to its original), so spaces moved together between
// servers still recognise each other.
func (s Backups) ImportSpace(ctx context.Context, spaces Spaces, path, name string, owner *auth.User, quota int64) (*Space, error) {
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
	return spaces.createFrom(ctx, name, owner, p)
}

// RestoreServer replaces the server and every space with a server backup (a
// zip made by CreateServer, or a raw single-file or Laravel-era database).
func (s Backups) RestoreServer(ctx context.Context, name string) error {
	if _, err := s.ServerBackup(name); err != nil {
		return err
	}
	src := s.ServerPath(name)
	var seed []byte
	if strings.HasSuffix(name, archiveExt) {
		arc, err := openArchive(src, s.tmpDir(), 0, s.Keys)
		if err != nil {
			return err
		}
		defer os.RemoveAll(arc.Dir)
		if arc.Manifest.Kind != kindServer {
			return ErrNotServerBackup
		}
		src = arc.Dir
		if raw, err := os.ReadFile(filepath.Join(arc.Dir, "keys", "server.ed25519")); err == nil {
			key, err := signing.Parse(raw)
			if err != nil {
				return err
			}
			seed = key.Seed()
		}
	}
	p, err := s.Store.PrepareServer(ctx, src)
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

func fileSizeOf(p string) int64 {
	info, err := os.Stat(p)
	if err != nil {
		return 0
	}
	return info.Size()
}
