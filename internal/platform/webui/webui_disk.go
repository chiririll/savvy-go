//go:build !embed

// Package webui holds the frontend built into the binary by release builds.
package webui

import "io/fs"

// FS is the built frontend (index.html, icons, build/), or nil when the binary
// was built without the embed tag and serves it from disk instead.
func FS() fs.FS { return nil }
