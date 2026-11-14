package backup

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
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
var archiveEntry = regexp.MustCompile(
	`^(manifest\.json|manifest\.sig|space\.sqlite|server\.sqlite|keys/server\.ed25519|spaces/[1-9][0-9]*\.sqlite)$`,
)

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
