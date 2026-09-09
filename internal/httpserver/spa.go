package httpserver

import (
	"encoding/json"
	"html"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/chiririll/savvy-plus/internal/version"
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
	if rel != "" && !strings.Contains(rel, "..") {
		full := filepath.Join(s.cfg.PublicDir, filepath.FromSlash(rel))
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			http.ServeFile(w, r, full)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(s.indexHTML()))
}

func (s *Server) indexHTML() string {
	js, css := s.viteAssets()
	var assets strings.Builder
	for _, href := range css {
		assets.WriteString(`    <link rel="stylesheet" href="` + html.EscapeString(href) + `">` + "\n")
	}
	for _, src := range js {
		assets.WriteString(`    <script type="module" src="` + html.EscapeString(src) + `"></script>` + "\n")
	}

	return `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Savvy</title>
    <link rel="icon" type="image/svg+xml" href="/favicon.svg">
    <link rel="icon" type="image/png" sizes="96x96" href="/favicon-96x96.png">
    <link rel="icon" type="image/x-icon" href="/favicon.ico">
    <link rel="apple-touch-icon" sizes="180x180" href="/apple-touch-icon.png">
    <link rel="manifest" href="/site.webmanifest">
    <meta name="theme-color" content="#ffffff">
    <meta name="app-version" content="` + html.EscapeString(version.Value) + `">
    <meta name="app-env" content="` + html.EscapeString(s.cfg.AppEnv) + `">
    <script>
        (function () {
            try {
                var stored = localStorage.getItem('theme');
                var theme = stored === 'dark' || stored === 'light'
                    ? stored
                    : (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
                var root = document.documentElement;
                root.classList.add(theme);
                root.style.colorScheme = theme;
                var meta = document.querySelector('meta[name="theme-color"]');
                if (meta) meta.setAttribute('content', theme === 'dark' ? '#1a1a1a' : '#ffffff');
            } catch (e) {}
        })();
    </script>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500;600&display=swap" rel="stylesheet">
` + assets.String() + `</head>
<body>
<div id="app"></div>
</body>
</html>
`
}

func (s *Server) viteAssets() (js []string, css []string) {
	raw, err := os.ReadFile(filepath.Join(s.cfg.PublicDir, "build", "manifest.json"))
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
