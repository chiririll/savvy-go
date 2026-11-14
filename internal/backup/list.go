package backup

import (
	"errors"
	"os"
	"path/filepath"
	"savvy-go/internal/signing"
	"sort"
	"time"
)

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

// ServerBackupWithNote finds a valid server backup with exactly this note that
// was written after since, or nil.
func (s Service) ServerBackupWithNote(note string, since time.Time) *Entry {
	list, err := s.ServerBackups()
	if err != nil {
		return nil
	}
	for _, b := range list {
		if !b.Valid || b.Note == nil || *b.Note != note {
			continue
		}
		if info, err := os.Stat(s.ServerPath(b.Filename)); err == nil && info.ModTime().After(since) {
			return &b
		}
	}
	return nil
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
