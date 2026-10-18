// Package config reads and writes the TOML config file, and is the only place that has it.
// Open reads the file; the program then asks for the sections it uses with Of and calls
// Finish. A Section is a table a package declares next to the code that reads it, with how
// it becomes the value the package uses; its handle gives that value. This package fills
// defaults, decodes, writes the file and replaces the value, so a change reaches whoever
// holds the handle.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/pelletier/go-toml/v2"
)

// header opens every written config file.
const header = `Go Savvy configuration, maintained by the application: it is rewritten on
every start (new keys appear with their defaults). Your values are kept;
comments and formatting are not.`

// ErrReadOnly is returned by Update when the config file cannot be written.
var ErrReadOnly = errors.New("the config file is read-only")

// FieldError is a value a person cannot be allowed to save, named by the key it is about so a
// form can show the message by the field. Edit passes the error of its check through, so a
// caller can tell it from a failure to write the file.
type FieldError struct{ Field, Message string }

func (e FieldError) Error() string { return e.Message }

// Open reads the TOML file at path. A missing file is an empty one; invalid TOML is an
// error. Nothing is written until Finish. The program then asks for the sections it uses
// with Of and calls Finish.
func Open(path string) (*Config, error) { return open(path) }

// Parse is a config read from TOML text, with no file behind it: nothing is ever written.
// For tests.
func Parse(text string) (*Config, error) {
	tree := map[string]any{}
	if err := toml.Unmarshal([]byte(text), &tree); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &Config{tree: tree}, nil
}

// Empty is a config with no file behind it: every section has its defaults, and nothing is
// ever written. For tests and for a program that wires packages without a config file.
func Empty() *Config { return &Config{tree: map[string]any{}} }

// Finish ends the asking for sections. It fails, writing nothing, when a section cannot read
// its table or decode it. Otherwise it logs the tables and keys no section reads, then
// writes the file, creating it on the first run; what no section reads is left out from now
// on. A file that cannot be written (read-only mount) is logged and otherwise ignored: the
// file is then not Writable. Call it once, after the last Of.
func (f *Config) Finish() error { return f.finish() }

// Writable reports whether the config file can be changed. False for a read-only mount and
// before Finish.
func (f *Config) Writable() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.writable
}

// Path is the file Open read.
func (f *Config) Path() string { return f.path }

// node is a slot without its types.
type node interface {
	name() string
	unknown(table map[string]any, found bool, children map[string]bool) []string
	render(over any) ([]byte, error)
}

// Config is the config file. It is safe for concurrent use.
type Config struct {
	mu       sync.Mutex // guards the fields below and serializes saves
	path     string
	tree     map[string]any
	nodes    []node
	errs     []error // sections whose table could not be read
	finished bool
	// writable is whether the config file could be written at Finish.
	writable bool
}

func open(path string) (*Config, error) {
	tree := map[string]any{}
	switch raw, err := os.ReadFile(path); {
	case err == nil:
		if err := toml.Unmarshal(raw, &tree); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
	case errors.Is(err, fs.ErrNotExist):
	default:
		return nil, fmt.Errorf("read config: %w", err)
	}
	return &Config{path: path, tree: tree}, nil
}

func (f *Config) finish() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.finished {
		return errors.New("config: Finish called twice")
	}
	if len(f.errs) > 0 {
		return fmt.Errorf("config %s: %w", f.path, errors.Join(f.errs...))
	}
	names := map[string]bool{}
	for _, n := range f.nodes {
		names[n.name()] = true
	}
	var unknown []string
	for _, n := range f.nodes {
		table, found := find(f.tree, n.name())
		unknown = append(unknown, n.unknown(table, found, childKeys(n.name(), names))...)
	}
	unknown = append(unknown, unclaimed(f.tree, "", names)...)
	if len(unknown) > 0 {
		sort.Strings(unknown)
		slog.Warn("ignoring unknown keys in the config file; they are left out when it is rewritten", "keys", unknown)
	}
	f.finished = true

	out, err := f.encode(nil, nil)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	// Write even when nothing changed: it is the probe for "can this process
	// save the config later" (a read-only mount or directory fails here).
	f.writable = true
	if err := writeFile(f.path, out); err != nil {
		f.writable = false
		slog.Warn(
			"could not write the config file; the values in it cannot be changed from the application",
			"path",
			f.path,
			"err",
			err,
		)
	}
	return nil
}

// find is the table at a dotted name.
func find(tree map[string]any, name string) (map[string]any, bool) {
	cur := tree
	for _, part := range strings.Split(name, ".") {
		next, ok := cur[part].(map[string]any)
		if !ok {
			return nil, false
		}
		cur = next
	}
	return cur, true
}

// childKeys are the names directly inside the table name that are sections of their own.
func childKeys(name string, sections map[string]bool) map[string]bool {
	out := map[string]bool{}
	for other := range sections {
		if rest, ok := strings.CutPrefix(other, name+"."); ok {
			out[strings.SplitN(rest, ".", 2)[0]] = true
		}
	}
	return out
}

// unclaimed are the keys of tree that no section reads: everything outside the sections, and
// the keys directly inside a table that only holds sections.
func unclaimed(tree map[string]any, prefix string, sections map[string]bool) []string {
	var out []string
	for k, v := range tree {
		path := prefix + k
		if sections[path] {
			continue
		}
		table, isTable := v.(map[string]any)
		if isTable && holdsSection(path, sections) {
			out = append(out, unclaimed(table, path+".", sections)...)
			continue
		}
		out = append(out, path)
	}
	return out
}

func holdsSection(path string, sections map[string]bool) bool {
	for s := range sections {
		if strings.HasPrefix(s, path+".") {
			return true
		}
	}
	return false
}

// encode is the text of the config file: the header, then the sections in the order they
// were asked for. over replaces the value of one section, as render says.
func (f *Config) encode(target node, over any) ([]byte, error) {
	var out []byte
	for _, line := range strings.Split(header, "\n") {
		out = append(out, strings.TrimRight("# "+line, " ")...)
		out = append(out, '\n')
	}
	for _, n := range f.nodes {
		var o any
		if n == target {
			o = over
		}
		text, err := n.render(o)
		if err != nil {
			return nil, err
		}
		if len(text) > 0 {
			out = append(append(out, '\n'), text...)
		}
	}
	return out, nil
}

// save writes the file with over as the value of target. The caller holds f.mu.
func (f *Config) save(target node, over any) error {
	out, err := f.encode(target, over)
	if err != nil {
		return err
	}
	return writeFile(f.path, out)
}

// writeFile replaces path atomically, creating its directory.
func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o775); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
