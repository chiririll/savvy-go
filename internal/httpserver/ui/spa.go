// Package ui serves what a browser opens: the frontend (static files, and index.html for every
// other path) and the API reference (Swagger UI and the OpenAPI document).
package ui

import (
	"github.com/go-chi/chi/v5"

	"bytes"
	"encoding/json"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path"
	"savvy-go/internal/appconfig"
	"savvy-go/internal/httpserver/httpx/routing"
	"strings"
)

type viteChunk struct {
	File    string   `json:"file"`
	CSS     []string `json:"css"`
	IsEntry bool     `json:"isEntry"`
	Src     string   `json:"src"`
}

func init() {
	routing.Register(func(d *routing.Deps) []routing.Mount {
		b := routing.Get[appconfig.Build](d)
		spa := NewSPA(b)
		return []routing.Mount{routing.Fallback(func(r chi.Router) { r.Get("/*", spa.ServeHTTP) })}
	})
}

// NewSPA is the frontend of the running binary: the copy built into it, or public/ next to
// the working directory in development builds. The version, env and instance id of b show in
// the page shell.
func NewSPA(b appconfig.Build) *SPA {
	return &SPA{Assets: Resolve(embeddedFS()), Version: b.Version, Env: b.Env, InstanceID: b.InstanceID}
}

// Resolve picks where the frontend comes from: the copy built into the
// binary, or public/ next to the working directory (development builds).
func Resolve(embedded fs.FS) fs.FS {
	if embedded != nil {
		return embedded
	}
	return os.DirFS("public")
}

// SPA serves the files of Assets.
type SPA struct {
	Assets  fs.FS
	Version string // the release, shown in the page shell
	Env     string // the build environment

	InstanceID string // the id of this install, shown in the page shell
}

// ServeHTTP serves the file asked for, or index.html for a client-side route.
func (h *SPA) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rel := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if rel != "" && rel != "index.html" && !strings.Contains(rel, "..") {
		if info, err := fs.Stat(h.Assets, rel); err == nil && !info.IsDir() {
			if strings.HasPrefix(rel, "build/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			http.ServeFileFS(w, r, h.Assets, rel)
			return
		}
		// A missing asset must 404: answering with index.html makes the
		// browser reject a stale chunk with a MIME error instead of a clean miss.
		if strings.HasPrefix(rel, "build/") || path.Ext(rel) != "" {
			http.NotFound(w, r)
			return
		}
	}

	page, err := h.renderIndex()
	if err != nil {
		http.Error(w, "index.html unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(page)
}

// indexData is what public/index.html is rendered with.
type indexData struct {
	Version    string
	Env        string
	InstanceID string
	JS         []string
	CSS        []string
}

// renderIndex renders the public/index.html template.
func (h *SPA) renderIndex() ([]byte, error) {
	tpl, err := template.ParseFS(h.Assets, "index.html")
	if err != nil {
		return nil, err
	}

	data := indexData{Version: h.Version, Env: h.Env, InstanceID: h.InstanceID}
	data.JS, data.CSS = h.viteAssets()

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (h *SPA) viteAssets() (js []string, css []string) {
	raw, err := fs.ReadFile(h.Assets, "build/manifest.json")
	if err != nil {
		raw, err = fs.ReadFile(h.Assets, "build/.vite/manifest.json")
	}
	if err != nil {
		return nil, nil
	}
	var manifest map[string]viteChunk
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, nil
	}
	for _, chunk := range manifest {
		if !chunk.IsEntry && chunk.Src != "resources/ts/main.tsx" {
			continue
		}
		if chunk.File != "" {
			js = append(js, "/build/"+chunk.File)
		}
		for _, c := range chunk.CSS {
			css = append(css, "/build/"+c)
		}
	}
	return js, css
}
