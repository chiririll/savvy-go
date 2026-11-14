// Package v1 holds what is needed to read a v1.0 space file: its frozen
// schema, and the conversion into the current one. A v1.0 file is never
// migrated in place; Convert writes a fresh file from the current base and
// copies the rows over.
package v1

import _ "embed"

// Schema is the v1.0 space schema (space/0001_initial), frozen. It is the
// shape of a v1.0 file and of what a Laravel restore produces.
//
//go:embed 0001_initial.sql
var Schema string

// Version is the version a v1.0 space file records in schema_migrations.
const Version = "space/0001_initial"
