package domain

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"savvy-go/internal/signing"
)

// A backup is a zip: the database files the store exported, manifest.json
// describing them (with each file's SHA-256) and manifest.sig, the server's
// signature of manifest.json. The signature tells a backup made here from one
// made elsewhere or edited; the hashes tie the files to the manifest.

const (
	manifestName  = "manifest.json"
	signatureName = "manifest.sig"
	// Zip limits against archive bombs: the unpacked size is also bounded by
	// the caller (the space quota plus room for the manifest).
	maxEntries = 10_000
)

// Archive entries a backup may hold. Anything else makes the archive invalid.
var archiveEntry = regexp.MustCompile(`^(manifest\.json|manifest\.sig|space\.sqlite|server\.sqlite|keys/server\.ed25519|spaces/[1-9][0-9]*\.sqlite)$`)

// Manifest describes a backup.
type Manifest struct {
	Format     int               `json:"format"`
	Kind       string            `json:"kind"` // "server" or "space"
	Store      string            `json:"store"`
	AppVersion string            `json:"app_version"`
	CreatedAt  string            `json:"created_at"`
	Note       string            `json:"note,omitempty"`
	SpaceUUID  string            `json:"space_uuid,omitempty"`
	SpaceName  string            `json:"space_name,omitempty"`
	KID        string            `json:"kid"`
	Files      map[string]string `json:"files"` // archive path -> SHA-256 hex
}

const (
	kindServer = "server"
	kindSpace  = "space"
)

// writeArchive zips files (archive path -> local path) with a manifest signed
// by key.
func writeArchive(dest string, m Manifest, files map[string]string, key *signing.Key) error {
	m.Files = map[string]string{}
	for name, local := range files {
		sum, err := fileSHA256(local)
		if err != nil {
			return err
		}
		m.Files[name] = sum
	}
	m.KID = key.KID()
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := dest + ".part"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(out)
	write := func(name string, r io.Reader) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, r)
		return err
	}
	err = write(manifestName, strings.NewReader(string(raw)))
	if err == nil {
		err = write(signatureName, strings.NewReader(base64.StdEncoding.EncodeToString(key.Sign(raw))))
	}
	for name, local := range files {
		if err != nil {
			break
		}
		var f *os.File
		if f, err = os.Open(local); err == nil {
			err = write(name, f)
			_ = f.Close()
		}
	}
	if cerr := zw.Close(); err == nil {
		err = cerr
	}
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}

func fileSHA256(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Signature is how far a backup can be trusted.
type Signature string

const (
	SignedHere    Signature = "own"      // signed by this server's key
	SignedTrusted Signature = "trusted"  // signed by a key this server trusts
	Unsigned      Signature = "unsigned" // no or an unknown signature
)

// ErrBadArchive is returned for an archive that is not a usable backup.
var ErrBadArchive = errors.New("the file is not a valid backup archive")

// openedArchive is a backup unpacked into a temporary directory.
type openedArchive struct {
	Dir       string
	Manifest  Manifest
	Signature Signature
}

// readManifest reads and checks the manifest of a backup without unpacking
// the databases.
func readManifest(p string, keys KeyRing) (*Manifest, Signature, error) {
	zr, err := zip.OpenReader(p)
	if err != nil {
		return nil, Unsigned, ErrBadArchive
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
			return nil, Unsigned, ErrBadArchive
		}
	}
	if raw == nil {
		return nil, Unsigned, ErrBadArchive
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil || (m.Kind != kindServer && m.Kind != kindSpace) {
		return nil, Unsigned, ErrBadArchive
	}
	return &m, keys.check(m.KID, raw, sig), nil
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
func openArchive(p, tmpRoot string, maxBytes int64, keys KeyRing) (*openedArchive, error) {
	m, sigState, err := readManifest(p, keys)
	if err != nil {
		return nil, err
	}
	zr, err := zip.OpenReader(p)
	if err != nil {
		return nil, ErrBadArchive
	}
	defer zr.Close()
	if len(zr.File) > maxEntries {
		return nil, ErrBadArchive
	}
	var total uint64
	for _, f := range zr.File {
		if !archiveEntry.MatchString(f.Name) || path.Clean(f.Name) != f.Name || !f.Mode().IsRegular() {
			return nil, fmt.Errorf("%w: unexpected entry %q", ErrBadArchive, f.Name)
		}
		total += f.UncompressedSize64
	}
	if maxBytes > 0 && total > uint64(maxBytes) {
		return nil, fmt.Errorf("%w: it unpacks to more than the space may hold", ErrBadArchive)
	}
	if err := os.MkdirAll(tmpRoot, 0o700); err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp(tmpRoot, "unpack-")
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*openedArchive, error) {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	var written int64
	for _, f := range zr.File {
		if f.Name == manifestName || f.Name == signatureName {
			continue
		}
		want, listed := m.Files[f.Name]
		if !listed {
			return fail(fmt.Errorf("%w: %s is not in the manifest", ErrBadArchive, f.Name))
		}
		dst := filepath.Join(dir, filepath.FromSlash(f.Name))
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			return fail(err)
		}
		n, sum, err := extract(f, dst, maxBytes-written, maxBytes > 0)
		if err != nil {
			return fail(err)
		}
		written += n
		if sum != want {
			return fail(fmt.Errorf("%w: %s does not match the manifest", ErrBadArchive, f.Name))
		}
	}
	for name := range m.Files {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name))); errors.Is(err, fs.ErrNotExist) {
			return fail(fmt.Errorf("%w: %s is missing", ErrBadArchive, name))
		}
	}
	return &openedArchive{Dir: dir, Manifest: *m, Signature: sigState}, nil
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
