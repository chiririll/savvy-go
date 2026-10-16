// Package dto converts domain types to HTTP response bodies.
// Keeping serialization here lets domain types stay free of JSON/HTTP concerns.
//
// Conventions: keys are camelCase; optional values are always present and null
// when unset; money is a JSON number at the scale the domain stored it with.
package dto

import (
	"time"

	"github.com/shopspring/decimal"
)

func init() {
	// Money goes over the wire as a JSON number, not a string. Values read from
	// storage already carry their currency's scale; computed totals are rounded
	// to it in the domain.
	decimal.MarshalJSONWithoutQuotes = true
}

// Map converts every element of in with fn. It never returns nil, so empty
// lists serialize as [].
func Map[T, R any](in []T, fn func(T) R) []R {
	out := make([]R, 0, len(in))
	for _, v := range in {
		out = append(out, fn(v))
	}
	return out
}

// ptr converts an optional domain value; nil stays nil.
func ptr[T, R any](v *T, fn func(T) R) *R {
	if v == nil {
		return nil
	}
	r := fn(*v)
	return &r
}

func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
