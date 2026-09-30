package httpserver

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"savvy-go/internal/version"
)

type viteChunk struct {
	File    string   `json:"file"`
	CSS     []string `json:"css"`
	IsEntry bool     `json:"isEntry"`
	Src     string   `json:"src"`
}

func (s *Server) spa(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rel := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if rel != "" && rel != "index.html" && !strings.Contains(rel, "..") {
		full := filepath.Join(s.cfg.PublicDir, filepath.FromSlash(rel))
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			http.ServeFile(w, r, full)
			return
		}
	}

	page, err := s.renderIndex()
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
	Version string
	Env     string
	JS      []string
	CSS     []string
}

// renderIndex renders the public/index.html template.
func (s *Server) renderIndex() ([]byte, error) {
	tpl, err := template.ParseFiles(filepath.Join(s.cfg.PublicDir, "index.html"))
	if err != nil {
		return nil, err
	}

	data := indexData{Version: version.Value, Env: version.Env}
	data.JS, data.CSS = s.viteAssets()

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *Server) viteAssets() (js []string, css []string) {
	raw, err := os.ReadFile(filepath.Join(s.cfg.PublicDir, "build", "manifest.json"))
	if err != nil {
		raw, err = os.ReadFile(filepath.Join(s.cfg.PublicDir, "build", ".vite", "manifest.json"))
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
