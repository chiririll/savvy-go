package httpserver

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"savvy-go/internal/auth"
)

// rezip rewrites an archive, letting edit change or drop entries and add new
// ones.
func rezip(
	t *testing.T,
	src string,
	edit func(name string, data []byte) ([]byte, bool),
	extra map[string][]byte,
) []byte {
	t.Helper()
	zr, err := zip.OpenReader(src)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range zr.File {
		r, _ := f.Open()
		data, _ := io.ReadAll(r)
		r.Close()
		if data, keep := edit(f.Name, data); keep {
			w, _ := zw.Create(f.Name)
			_, _ = w.Write(data)
		}
	}
	for name, data := range extra {
		w, _ := zw.Create(name)
		_, _ = w.Write(data)
	}
	_ = zw.Close()
	return buf.Bytes()
}

// How a tampered archive shows through the API: an edited manifest
// is listed "unsigned" but restores, an inconsistent archive is a 422. The
// archive checks themselves are tested in internal/backup.
func TestTamperedArchives(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	b := createdBackup(t, a.do("POST", spacePath(a.space.ID, "/backups"), map[string]any{}, sess.Token, sess.CSRF))
	orig := a.svc.Backups.SpacePath(a.space.ID, b["filename"].(string))

	restore := func(content []byte) (int, map[string]any) {
		up := createdBackup(t, a.upload(spacePath(a.space.ID, "/backups/upload"), "x.zip", content, nil, sess))
		res := a.do("POST", spacePath(a.space.ID, "/backups/"+up["filename"].(string)+"/restore"), nil, sess.Token, sess.CSRF)
		body := decodeJSON(t, res)
		body["listed"] = up
		return res.StatusCode, body
	}

	// The note changed: still restorable, but no longer signed by us.
	edited := rezip(t, orig, func(name string, data []byte) ([]byte, bool) {
		if name == "manifest.json" {
			var m map[string]any
			_ = json.Unmarshal(data, &m)
			m["note"] = "edited"
			data, _ = json.Marshal(m)
		}
		return data, true
	}, nil)
	code, body := restore(edited)
	if listed := body["listed"].(map[string]any); listed["status"] != "unsigned" || code != http.StatusOK {
		t.Fatalf("edited manifest: listed %v restore %d %v", listed, code, body)
	}

	// The database changed: it no longer matches the manifest.
	swapped := rezip(t, orig, func(name string, data []byte) ([]byte, bool) {
		if name == "space.sqlite" {
			data = append(bytes.Clone(data), 0)
		}
		return data, true
	}, nil)
	if code, body := restore(swapped); code != http.StatusUnprocessableEntity ||
		!strings.Contains(body["message"].(string), "does not match") {
		t.Fatalf("swapped database: %d %v", code, body)
	}

	bad := rezip(
		t,
		orig,
		func(name string, data []byte) ([]byte, bool) { return data, true },
		map[string][]byte{"../escape.sqlite": []byte("x")},
	)
	if code, body := restore(bad); code != http.StatusUnprocessableEntity {
		t.Fatalf("unexpected entry: %d %v", code, body)
	}
}
