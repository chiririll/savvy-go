// Package locator is the registry the process wires itself with. A package declares how its
// service is built and what it needs by asking for other services; the locator builds each
// one once, when first asked, and closes them in reverse order of construction. It knows no
// service by name: a service is found by its Go type, so give a service its own type
// (a struct, or a named type for a plain string) rather than registering a bare string.
//
// Build all services at start with [Locator.Build], so a missing or broken provider fails
// the start instead of a request. A locator is not for request paths: handlers take what
// they need from it when their routes are built, and keep it.
package locator

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"
)

var (
	// ErrMissing is returned when nothing provides the type asked for.
	ErrMissing = errors.New("no provider")
	// ErrCycle is returned when a service needs itself, directly or through others.
	ErrCycle = errors.New("dependency cycle")
)

type state int

const (
	idle state = iota
	building
	done
)

type entry struct {
	build func(*Locator) (any, error)
	state state
	value any
	err   error
}

// Locator holds the providers and the services built from them.
type Locator struct {
	mu      sync.Mutex
	entries map[reflect.Type]*entry
	closers []func(context.Context) error
	closed  bool
}

// New is an empty locator.
func New() *Locator { return &Locator{entries: map[reflect.Type]*entry{}} }

func typeOf[T any]() reflect.Type { return reflect.TypeOf((*T)(nil)).Elem() }

// Provide registers how to build a T. build runs once, on the first Get of T or on Build;
// it asks the locator it is given for what it needs. Two providers for one type is a wiring
// bug and panics.
func Provide[T any](l *Locator, build func(*Locator) (T, error)) {
	t := typeOf[T]()
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.entries[t]; ok {
		panic(fmt.Sprintf("locator: %v is provided twice", t))
	}
	l.entries[t] = &entry{build: func(l *Locator) (any, error) { return build(l) }}
}

// Set makes v the T, replacing any provider or value: how a test swaps a service for a fake,
// and how the process hands in what it made before the locator (the config file).
func Set[T any](l *Locator, v T) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries[typeOf[T]()] = &entry{state: done, value: v}
}

// Get is the T, built if need be. It panics when it cannot be built, because that is a
// wiring bug; use TryGet inside a provider that wants to report it.
func Get[T any](l *Locator) T {
	v, err := TryGet[T](l)
	if err != nil {
		panic(fmt.Sprintf("locator: %v", err))
	}
	return v
}

// TryGet is Get that returns the error. The error names the services that were being built,
// innermost last.
func TryGet[T any](l *Locator) (T, error) {
	t := typeOf[T]()
	v, err := l.resolve(t)
	if err != nil {
		var zero T
		return zero, err
	}
	return v.(T), nil
}

func (l *Locator) resolve(t reflect.Type) (any, error) {
	l.mu.Lock()
	e, ok := l.entries[t]
	if !ok {
		l.mu.Unlock()
		return nil, fmt.Errorf("%v: %w", t, ErrMissing)
	}
	switch e.state {
	case done:
		l.mu.Unlock()
		return e.value, e.err
	case building:
		l.mu.Unlock()
		return nil, fmt.Errorf("%v: %w", t, ErrCycle)
	}
	e.state = building
	l.mu.Unlock()

	value, err := e.build(l)

	l.mu.Lock()
	defer l.mu.Unlock()
	e.state = done
	if err != nil {
		e.err = fmt.Errorf("%v: %w", t, err)
		return nil, e.err
	}
	e.value = value
	return value, nil
}

// OnClose registers a function Close runs for the service being built, after those of the
// services built later. Call it from a provider.
func OnClose(l *Locator, fn func(context.Context) error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closers = append(l.closers, fn)
}

// Build builds every registered service and returns the first error. Call it once at start,
// before serving.
func (l *Locator) Build() error {
	l.mu.Lock()
	types := make([]reflect.Type, 0, len(l.entries))
	for t := range l.entries {
		types = append(types, t)
	}
	l.mu.Unlock()
	// Map order is random; sort so a failure is the same on every run.
	sortTypes(types)
	for _, t := range types {
		if _, err := l.resolve(t); err != nil {
			return err
		}
	}
	return nil
}

// Close runs the OnClose functions in reverse order of registration, which is the reverse
// of construction, and returns their errors joined. Calling it again does nothing.
func (l *Locator) Close(ctx context.Context) error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	closers := l.closers
	l.mu.Unlock()

	var errs []error
	for i := len(closers) - 1; i >= 0; i-- {
		errs = append(errs, closers[i](ctx))
	}
	return errors.Join(errs...)
}

func sortTypes(types []reflect.Type) {
	sort.Slice(types, func(i, j int) bool { return types[i].String() < types[j].String() })
}
