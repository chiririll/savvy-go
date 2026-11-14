package backup

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"savvy-go/internal/signing"
)

// ErrBadArchive is returned for an archive that is not a usable backup.
var ErrBadArchive = errors.New("the file is not a valid backup archive")

// openedArchive is a backup unpacked into a temporary directory.
type openedArchive struct {
	Dir       string
	Manifest  Manifest
	Signature signing.Signature
}

// readManifest reads and checks the manifest of a backup without unpacking
// the databases.
func readManifest(p string, keys signing.KeyRing) (*Manifest, signing.Signature, error) {
	zr, err := zip.OpenReader(p)
	if err != nil {
		return nil, signing.Unsigned, ErrBadArchive
	}
	defer zr.Close()
	var raw, sig []byte
	for _, f := range zr.File {
		switch f.Name {
		case manifestName:
			raw, err = readEntry(f, 1<<20)
		case signatureName:
			sig, err = readEntry(f, 1<<10)
		}
		if err != nil {
			return nil, signing.Unsigned, ErrBadArchive
		}
	}
	if raw == nil {
		return nil, signing.Unsigned, ErrBadArchive
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil || (m.Kind != kindServer && m.Kind != kindSpace) {
		return nil, signing.Unsigned, ErrBadArchive
	}
	return &m, keys.Check(m.KID, raw, sig), nil
}

func readEntry(f *zip.File, limit int64) ([]byte, error) {
	r, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err == nil && int64(len(b)) > limit {
		err = errors.New("entry too large")
	}
	return b, err
}

// openArchive unpacks a backup into a new directory under tmpRoot after
// checking every entry name, the total size and the file hashes. maxBytes
// bounds the unpacked size.
func openArchive(p, tmpRoot string, maxBytes int64, keys signing.KeyRing) (*openedArchive, error) {
	m, sigState, err := readManifest(p, keys)
	if err != nil {
		return nil, err
	}
	zr, err := zip.OpenReader(p)
	if err != nil {
		return nil, ErrBadArchive
	}
	defer zr.Close()
	if err := checkEntries(zr.File, maxBytes); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(tmpRoot, 0o700); err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp(tmpRoot, "unpack-")
	if err != nil {
		return nil, err
	}
	if err := unpack(zr.File, m, dir, maxBytes); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	return &openedArchive{Dir: dir, Manifest: *m, Signature: sigState}, nil
}

// checkEntries rejects an archive with too many entries, an entry that is not
// a plain file with a clean name, or a total size above maxBytes.
func checkEntries(files []*zip.File, maxBytes int64) error {
	if len(files) > maxEntries {
		return ErrBadArchive
	}
	var total uint64
	for _, f := range files {
		if !archiveEntry.MatchString(f.Name) || path.Clean(f.Name) != f.Name || !f.Mode().IsRegular() {
			return fmt.Errorf("%w: unexpected entry %q", ErrBadArchive, f.Name)
		}
		total += f.UncompressedSize64
	}
	if maxBytes > 0 && total > uint64(maxBytes) {
		return fmt.Errorf("%w: it unpacks to more than the space may hold", ErrBadArchive)
	}
	return nil
}

// unpack writes every listed entry into dir and checks it against the manifest.
func unpack(files []*zip.File, m *Manifest, dir string, maxBytes int64) error {
	var written int64
	for _, f := range files {
		if f.Name == manifestName || f.Name == signatureName {
			continue
		}
		want, listed := m.Files[f.Name]
		if !listed {
			return fmt.Errorf("%w: %s is not in the manifest", ErrBadArchive, f.Name)
		}
		dst := filepath.Join(dir, filepath.FromSlash(f.Name))
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			return err
		}
		n, sum, err := extract(f, dst, maxBytes-written, maxBytes > 0)
		if err != nil {
			return err
		}
		written += n
		if sum != want {
			return fmt.Errorf("%w: %s does not match the manifest", ErrBadArchive, f.Name)
		}
	}
	for name := range m.Files {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name))); errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%w: %s is missing", ErrBadArchive, name)
		}
	}
	return nil
}

// extract writes one entry, never more than limit bytes when bounded (the
// declared size can lie).
func extract(f *zip.File, dst string, limit int64, bounded bool) (int64, string, error) {
	r, err := f.Open()
	if err != nil {
		return 0, "", ErrBadArchive
	}
	defer r.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return 0, "", err
	}
	defer out.Close()
	h := sha256.New()
	var src io.Reader = r
	if bounded {
		src = io.LimitReader(r, limit+1)
	}
	n, err := io.Copy(io.MultiWriter(out, h), src)
	if err != nil {
		return n, "", ErrBadArchive
	}
	if bounded && n > limit {
		return n, "", fmt.Errorf("%w: it unpacks to more than the space may hold", ErrBadArchive)
	}
	return n, hex.EncodeToString(h.Sum(nil)), nil
}
