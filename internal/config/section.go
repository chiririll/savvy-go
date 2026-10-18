package config

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/pelletier/go-toml/v2"
)

// Section describes one table of the config file: where it is, what it holds and how it
// becomes the value its package uses. A package declares the sections it uses next to the
// code that reads them; this package never sees their fields. A package may declare
// several, and a section may be inside another one ("mailer" and "mailer.smtp").
//
// T is the table: a flat struct of strings, numbers and booleans with `toml` tags, and
// `comment` tags for the lines written above the keys. It holds no tables, slices or maps.
// When T implements [Defaulter], its defaults fill the keys the file lacks and the section
// is always written; otherwise a key the file lacks is the zero value and the section is
// written only when the file has it.
//
// V is what the package works with, made by Decode.
type Section[T, V any] struct {
	// Name is the table name, dotted for a nested table.
	Name string
	// Decode turns the table into the value the package uses: units, trimming, checks. An
	// error stops Finish, so a value the program cannot run with stops it at start, and is
	// the error of Update. Nil is only allowed when T and V are the same type.
	Decode func(T) (V, error)
}

// Optional is a section that is absent unless the file has the table, such as one of several
// ways to do the same thing. An empty table is present. A table type with defaults is always
// present: its defaults are written like those of any other section.
type Optional[T, V any] Section[T, V]

// Defaulter is a table type that has defaults. Implement it on the value, not the pointer.
type Defaulter[T any] interface{ Defaults() T }

// Ref is the handle of a Section: what the package keeps and calls Get on when it needs the
// value.
type Ref[V any] struct{ src source[V] }

// Get is the section as it is now. A nil Ref gives the zero value.
func (r *Ref[V]) Get() V {
	if r == nil {
		var zero V
		return zero
	}
	v, _ := r.src.value()
	return v
}

// Subscribe calls fn with the new value after each change of the section. Most packages read
// Get when they need the value and do not need it; it is for those that build something
// from the section once. Call the returned function to stop.
func (r *Ref[V]) Subscribe(fn func(V)) (cancel func()) {
	return r.src.subscribe(func(v V, _ bool) { fn(v) })
}

// OptRef is the handle of an Optional section.
type OptRef[V any] struct{ src source[V] }

// Get is the section and whether the file has it. A nil OptRef is absent.
func (r *OptRef[V]) Get() (V, bool) {
	if r == nil {
		var zero V
		return zero, false
	}
	return r.src.value()
}

// Subscribe is Ref.Subscribe; present is false after Remove.
func (r *OptRef[V]) Subscribe(fn func(v V, present bool)) (cancel func()) {
	return r.src.subscribe(fn)
}

// Fixed is a Ref to v that is not in any file and never changes, for tests and for a
// program that wires a package without a config file.
func Fixed[V any](v V) *Ref[V] { return &Ref[V]{src: fixed[V]{v}} }

// Of is the handle of s in c. Asking again for the same section gives
// another handle to the same value. A table the section cannot read is reported by Finish.
// It panics when a new section is asked for after Finish, or when the name was
// used for another type or kind: all mistakes in how the program is put together, found at
// start.
func Of[T, V any](c *Config, s Section[T, V]) *Ref[V] {
	return &Ref[V]{src: register(c, s, false)}
}

// OfOptional is Of for an Optional section.
func OfOptional[T, V any](c *Config, s Optional[T, V]) *OptRef[V] {
	return &OptRef[V]{src: register(c, Section[T, V](s), true)}
}

// Update changes the section with change, decodes it, then writes the file. It fails with
// ErrReadOnly when the file cannot be written, and with the error of Decode. Nothing is kept
// on failure. It panics when the section was never asked for with Of.
func (s Section[T, V]) Update(c *Config, change func(*T)) error {
	return lookup[T, V](c, s.Name, false).update(change)
}

// Update is Section.Update, adding the section when it is absent.
func (s Optional[T, V]) Update(c *Config, change func(*T)) error {
	return lookup[T, V](c, s.Name, true).update(change)
}

// Remove takes the section out of the file. Absent already is not an error.
func (s Optional[T, V]) Remove(c *Config) error {
	return lookup[T, V](c, s.Name, true).remove()
}

// source is where a handle reads its value.
type source[V any] interface {
	value() (V, bool)
	subscribe(fn func(V, bool)) func()
}

type fixed[V any] struct{ v V }

func (f fixed[V]) value() (V, bool)             { return f.v, true }
func (fixed[V]) subscribe(func(V, bool)) func() { return func() {} }

func register[T, V any](f *Config, s Section[T, V], optional bool) *slot[T, V] {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, n := range f.nodes {
		if n.name() != s.Name {
			continue
		}
		sl, ok := n.(*slot[T, V])
		if !ok || sl.optional != optional {
			panic("config: section " + s.Name + " is used as another type or kind")
		}
		return sl
	}
	if f.finished {
		panic("config: section " + s.Name + " asked for after Finish")
	}
	sl := &slot[T, V]{file: f, nameStr: s.Name, decode: s.Decode, optional: optional}
	if sl.decode == nil {
		var zero T
		if _, ok := any(zero).(V); !ok {
			panic("config: section " + s.Name + " needs Decode")
		}
		sl.decode = func(t T) (V, error) { return any(t).(V), nil }
	}
	var zero T
	if d, ok := any(zero).(Defaulter[T]); ok {
		sl.def, sl.hasDefaults = d.Defaults(), true
	}
	table, found := find(f.tree, s.Name)
	if err := sl.load(table, found); err != nil {
		f.errs = append(f.errs, err)
	}
	f.nodes = append(f.nodes, sl)
	return sl
}

func lookup[T, V any](f *Config, name string, optional bool) *slot[T, V] {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, n := range f.nodes {
		if n.name() == name {
			if sl, ok := n.(*slot[T, V]); ok && sl.optional == optional {
				return sl
			}
			break
		}
	}
	panic("config: section " + name + " changed but not asked for as this type or kind")
}

// state is a section that is present.
type state[T, V any] struct {
	raw T
	val V
	// inFile is whether the file has the table: a section without defaults is written only
	// then.
	inFile bool
}

// slot holds one section.
type slot[T, V any] struct {
	file        *Config
	nameStr     string
	decode      func(T) (V, error)
	def         T
	hasDefaults bool
	optional    bool
	p           atomic.Pointer[state[T, V]] // nil: absent

	subMu sync.Mutex
	subs  map[int]func(V, bool)
	next  int
}

func (s *slot[T, V]) name() string { return s.nameStr }

func (s *slot[T, V]) value() (V, bool) {
	if st := s.p.Load(); st != nil {
		return st.val, true
	}
	var zero V
	return zero, false
}

func (s *slot[T, V]) subscribe(fn func(V, bool)) func() {
	s.subMu.Lock()
	defer s.subMu.Unlock()
	if s.subs == nil {
		s.subs = map[int]func(V, bool){}
	}
	id := s.next
	s.next++
	s.subs[id] = fn
	return func() {
		s.subMu.Lock()
		defer s.subMu.Unlock()
		delete(s.subs, id)
	}
}

func (s *slot[T, V]) notify(v V, present bool) {
	s.subMu.Lock()
	fns := make([]func(V, bool), 0, len(s.subs))
	for _, fn := range s.subs {
		fns = append(fns, fn)
	}
	s.subMu.Unlock()
	for _, fn := range fns {
		fn(v, present)
	}
}

// load sets the value from the table of the file. Keys it does not know are ignored; see
// unknown.
func (s *slot[T, V]) load(table map[string]any, found bool) error {
	if !found && s.optional && !s.hasDefaults {
		return nil
	}
	raw := s.def
	if found {
		text, err := toml.Marshal(table)
		if err != nil {
			return err
		}
		if err := toml.Unmarshal(text, &raw); err != nil {
			return fmt.Errorf("[%s]: %w", s.nameStr, err)
		}
	}
	val, err := s.decode(raw)
	if err != nil {
		return fmt.Errorf("[%s]: %w", s.nameStr, err)
	}
	s.p.Store(&state[T, V]{raw: raw, val: val, inFile: found && !onlyTables(table)})
	return nil
}

// onlyTables is whether a table has keys and all of them are tables: then the file names it
// only as the parent of its sections.
func onlyTables(table map[string]any) bool {
	for _, v := range table {
		if _, ok := v.(map[string]any); !ok {
			return false
		}
	}
	return len(table) > 0
}

// unknown are the keys of the table that mean nothing. children are the keys of the table
// that are sections of their own.
func (s *slot[T, V]) unknown(table map[string]any, found bool, children map[string]bool) []string {
	if !found {
		return nil
	}
	own := make(map[string]any, len(table))
	for k, v := range table {
		if !children[k] {
			own[k] = v
		}
	}
	raw, err := toml.Marshal(own)
	if err != nil {
		return nil
	}
	dec := toml.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var v T
	var miss *toml.StrictMissingError
	if err := dec.Decode(&v); !errors.As(err, &miss) {
		return nil
	}
	var out []string
	for _, e := range miss.Errors {
		out = append(out, s.nameStr+"."+strings.Join(e.Key(), "."))
	}
	return out
}

// render is the text of the section: the current value, or over when it is a *T. A nil *T
// is the section taken out. Nothing is rendered for an absent section, for one without
// defaults that the file lacks, and for a table without keys the file lacks (a section that
// only holds others).
func (s *slot[T, V]) render(over any) ([]byte, error) {
	var raw T
	var inFile bool
	switch o := over.(type) {
	case nil:
		st := s.p.Load()
		if st == nil || (!st.inFile && !s.hasDefaults) {
			return nil, nil
		}
		raw, inFile = st.raw, st.inFile
	case *T:
		if o == nil {
			return nil, nil
		}
		raw, inFile = *o, true
	default:
		panic("config: wrong value for section " + s.nameStr)
	}
	body, err := toml.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("[%s]: %w", s.nameStr, err)
	}
	if len(bytes.TrimSpace(body)) == 0 && !inFile {
		return nil, nil
	}
	return append([]byte("["+s.nameStr+"]\n"), body...), nil
}

func (s *slot[T, V]) update(change func(*T)) error {
	f := s.file
	f.mu.Lock()
	if !f.writable {
		f.mu.Unlock()
		return ErrReadOnly
	}
	next := s.def
	if st := s.p.Load(); st != nil {
		next = st.raw
	}
	change(&next)
	val, err := s.decode(next)
	if err != nil {
		f.mu.Unlock()
		return err
	}
	if err := f.save(s, &next); err != nil {
		f.mu.Unlock()
		return fmt.Errorf("save %s: %w", f.path, err)
	}
	s.p.Store(&state[T, V]{raw: next, val: val, inFile: true})
	f.mu.Unlock()
	s.notify(val, true)
	return nil
}

func (s *slot[T, V]) remove() error {
	f := s.file
	f.mu.Lock()
	if !f.writable {
		f.mu.Unlock()
		return ErrReadOnly
	}
	if s.p.Load() == nil {
		f.mu.Unlock()
		return nil
	}
	if err := f.save(s, (*T)(nil)); err != nil {
		f.mu.Unlock()
		return fmt.Errorf("save %s: %w", f.path, err)
	}
	s.p.Store(nil)
	f.mu.Unlock()
	var zero V
	s.notify(zero, false)
	return nil
}
