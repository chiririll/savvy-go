// Package v1 holds what is needed to read a v1.0 space file: its frozen
// schema. The conversion into the current schema builds on it.
package v1

import _ "embed"

// Schema is the v1.0 space schema (space/0001_initial), frozen. It is the
// shape of a v1.0 file and of what a Laravel restore produces.
//
//go:embed 0001_initial.sql
var Schema string
