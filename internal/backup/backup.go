package backup

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"savvy-go/internal/signing"
	"savvy-go/internal/store"
	"savvy-go/internal/transfers"
)

// Backups of the whole server live in <Dir>/server, backups of a space in
// <Dir>/spaces/<id>. A server backup holds every database and the signing
// key, so only server admins see them; a space backup holds one space and is
// managed by that space's admins.

const (
	archiveExt = ".savvy.zip"
	// zipExt is what a listed backup must end with, so older ".zip" names stay.
	zipExt = ".zip"
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
	// AppVersion is recorded in every backup manifest.
	AppVersion string
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
