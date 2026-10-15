package backup

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"savvy-go/internal/signing"
)

type archiveFixture struct {
	t    *testing.T
	key  *signing.Key
	keys signing.KeyRing
	path string // a valid space backup signed by key
}

func newArchiveFixture(t *testing.T) *archiveFixture {
	t.Helper()
	key, err := signing.Generate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &archiveFixture{t: t, key: key, keys: signing.KeyRing{Holder: signing.NewHolder(key)}}
	db := filepath.Join(t.TempDir(), "space.sqlite")
	if err := os.WriteFile(db, []byte("space data"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.path = filepath.Join(t.TempDir(), "b.zip")
	m := Manifest{Kind: kindSpace, SpaceUUID: "u-1", SpaceName: "Home"}
	if err := writeArchive(f.path, m, map[string]string{"space.sqlite": db}, key); err != nil {
		t.Fatal(err)
	}
	return f
}

// rewrite copies the fixture archive through edit, which may change or drop
// (nil) each entry, and adds extra entries. Hashes and signature are kept as
// they were, so any edit makes the archive inconsistent.
func (f *archiveFixture) rewrite(edit func(name string, data []byte) []byte, extra map[string][]byte) string {
	f.t.Helper()
	zr, err := zip.OpenReader(f.path)
	if err != nil {
		f.t.Fatal(err)
	}
	defer zr.Close()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	put := func(name string, data []byte) {
		w, err := zw.Create(name)
		if err != nil {
			f.t.Fatal(err)
		}
		_, _ = w.Write(data)
	}
	for _, e := range zr.File {
		r, _ := e.Open()
		data := new(bytes.Buffer)
		_, _ = data.ReadFrom(r)
		_ = r.Close()
		if edit != nil {
			if out := edit(e.Name, data.Bytes()); out != nil {
				put(e.Name, out)
			}
			continue
		}
		put(e.Name, data.Bytes())
	}
	for name, data := range extra {
		put(name, data)
	}
	if err := zw.Close(); err != nil {
		f.t.Fatal(err)
	}
	out := filepath.Join(f.t.TempDir(), "edited.zip")
	if err := os.WriteFile(out, buf.Bytes(), 0o600); err != nil {
		f.t.Fatal(err)
	}
	return out
}

func TestArchiveRoundTrip(t *testing.T) {
	f := newArchiveFixture(t)
	arc, err := openArchive(f.path, t.TempDir(), 0, f.keys)
	if err != nil {
		t.Fatal(err)
	}
	if arc.Signature != signing.SignedHere {
		t.Fatalf("signature %q", arc.Signature)
	}
	if arc.Manifest.Kind != kindSpace || arc.Manifest.SpaceUUID != "u-1" || arc.Manifest.KID != f.key.KID() {
		t.Fatalf("manifest %+v", arc.Manifest)
	}
	got, err := os.ReadFile(filepath.Join(arc.Dir, "space.sqlite"))
	if err != nil || string(got) != "space data" {
		t.Fatalf("unpacked %q %v", got, err)
	}
}

func TestArchiveSignatureStates(t *testing.T) {
	f := newArchiveFixture(t)

	t.Run("edited manifest is unsigned but readable", func(t *testing.T) {
		edited := f.rewrite(func(name string, data []byte) []byte {
			if name == manifestName {
				var m map[string]any
				_ = json.Unmarshal(data, &m)
				m["note"] = "edited"
				data, _ = json.Marshal(m)
			}
			return data
		}, nil)
		arc, err := openArchive(edited, t.TempDir(), 0, f.keys)
		if err != nil {
			t.Fatal(err)
		}
		if arc.Signature != signing.Unsigned {
			t.Fatalf("signature %q", arc.Signature)
		}
	})

	t.Run("missing signature file is unsigned", func(t *testing.T) {
		unsigned := f.rewrite(func(name string, data []byte) []byte {
			if name == signatureName {
				return nil
			}
			return data
		}, nil)
		arc, err := openArchive(unsigned, t.TempDir(), 0, f.keys)
		if err != nil || arc.Signature != signing.Unsigned {
			t.Fatalf("%v %v", arc, err)
		}
	})

	t.Run("foreign key is trusted only when listed", func(t *testing.T) {
		other := signing.KeyRing{Holder: signing.NewHolder(mustKey(t))}
		if arc, err := openArchive(f.path, t.TempDir(), 0, other); err != nil || arc.Signature != signing.Unsigned {
			t.Fatalf("unknown key: %v %v", arc, err)
		}
		other.Trusted = func(kid string) ed25519.PublicKey {
			if kid == f.key.KID() {
				return f.key.Public()
			}
			return nil
		}
		if arc, err := openArchive(f.path, t.TempDir(), 0, other); err != nil || arc.Signature != signing.SignedTrusted {
			t.Fatalf("trusted key: %v %v", arc, err)
		}
	})
}

func mustKey(t *testing.T) *signing.Key {
	t.Helper()
	k, err := signing.Generate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestArchiveRefusesInconsistentContent(t *testing.T) {
	f := newArchiveFixture(t)
	cases := map[string]struct {
		path string
		want string
	}{
		"changed database": {
			f.rewrite(func(name string, data []byte) []byte {
				if name == "space.sqlite" {
					return append(bytes.Clone(data), 0)
				}
				return data
			}, nil),
			"does not match",
		},
		"database removed": {
			f.rewrite(func(name string, data []byte) []byte {
				if name == "space.sqlite" {
					return nil
				}
				return data
			}, nil),
			"is missing",
		},
		"file not in manifest": {
			f.rewrite(nil, map[string][]byte{"server.sqlite": []byte("x")}),
			"not in the manifest",
		},
		"no manifest": {
			f.rewrite(func(name string, data []byte) []byte {
				if name == manifestName {
					return nil
				}
				return data
			}, nil),
			"",
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			_, err := openArchive(c.path, dir, 0, f.keys)
			if !errors.Is(err, ErrBadArchive) || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("got %v", err)
			}
			assertEmpty(t, dir)
		})
	}
}

func TestArchiveRefusesUnexpectedEntryNames(t *testing.T) {
	f := newArchiveFixture(t)
	for _, name := range []string{"../escape.sqlite", "/abs.sqlite", "spaces/../x.sqlite", "spaces/0.sqlite", "extra.txt", "keys/other.ed25519"} {
		t.Run(name, func(t *testing.T) {
			bad := f.rewrite(nil, map[string][]byte{name: []byte("x")})
			root := t.TempDir()
			dir := filepath.Join(root, "unpack")
			if _, err := openArchive(bad, dir, 0, f.keys); !errors.Is(err, ErrBadArchive) {
				t.Fatalf("got %v", err)
			}
			assertEmpty(t, dir)
			if entries, _ := filepath.Glob(filepath.Join(root, "*escape*")); len(entries) != 0 {
				t.Fatalf("an entry escaped: %v", entries)
			}
		})
	}
}

func TestArchiveSizeLimit(t *testing.T) {
	f := newArchiveFixture(t)
	if _, err := openArchive(f.path, t.TempDir(), 1<<20, f.keys); err != nil {
		t.Fatalf("within the limit: %v", err)
	}
	dir := t.TempDir()
	if _, err := openArchive(f.path, dir, 1, f.keys); !errors.Is(err, ErrBadArchive) {
		t.Fatalf("over the limit: %v", err)
	}
	assertEmpty(t, dir)
}

func TestValidName(t *testing.T) {
	for name, want := range map[string]bool{
		"20260101-120000.zip": true,
		"":                    false,
		"../x.zip":            false,
		"a/b.zip":             false,
		".hidden.zip":         false,
		"b.sqlite":            false,
	} {
		if got := validName(name); got != want {
			t.Errorf("validName(%q) = %v", name, got)
		}
	}
}

// assertEmpty checks that a refused archive left nothing unpacked.
func assertEmpty(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("left behind: %v", entries)
	}
}
