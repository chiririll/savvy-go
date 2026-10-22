//go:build !embed

package ui

import "io/fs"

// embeddedFS is the built frontend (index.html, icons, build/), or nil when the binary
// was built without the embed tag and serves it from disk instead.
func embeddedFS() fs.FS { return nil }
