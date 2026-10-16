//go:build embed

// Package webui holds the frontend built into the binary by release builds.
package webui

import (
	"embed"
	"io/fs"
)

// all: keeps build/.vite/manifest.json, which sits in a dot directory.
//
//go:embed all:dist
var dist embed.FS

// FS is the built frontend (index.html, icons, build/), or nil when the binary
// was built without the embed tag and serves it from disk instead.
func FS() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil
	}
	return sub
}
