package filter

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseSearchReadsNumbers(t *testing.T) {
	for in, want := range map[string][]amount{
		"1879":                 {{1879, 0}},
		"1 879":                {{1879, 0}},
		"1 879":                {{1879, 0}},
		"12 345.60":            {{123456, 1}},
		"18.79":                {{1879, 2}},
		"18,79":                {{1879, 2}},
		"1879.00":              {{1879, 0}},
		"1879.50":              {{18795, 1}},
		"1,879":                {{1879, 3}, {1879, 0}},
		"1,879.50":             {{18795, 1}},
		"1.879,50":             {{18795, 1}},
		"1,879,000":            {{1879000, 0}},
		"2026-01-02":           nil,
		"abc":                  nil,
		"1.2345678901234":      nil, // more fraction digits than any currency has
		"99999999999999999999": nil, // beyond money.MaxMinor
	} {
		terms := parseSearch(in)
		if len(terms) != 1 {
			t.Errorf("%q: %d terms", in, len(terms))
			continue
		}
		if !reflect.DeepEqual(terms[0].Amounts, want) {
			t.Errorf("%q: amounts %v, want %v", in, terms[0].Amounts, want)
		}
	}
}

func TestParseSearchBoundsTheQuery(t *testing.T) {
	if n := len(parseSearch(strings.Repeat("a ", 100))); n != maxSearchTerms {
		t.Errorf("%d terms, want %d", n, maxSearchTerms)
	}
	long := parseSearch(strings.Repeat("я", 5000))
	if len(long) != 1 || len([]rune(long[0].Text)) != maxSearchRunes {
		t.Errorf("long query not cut to %d runes", maxSearchRunes)
	}
	if got := parseSearch("  \t "); len(got) != 0 {
		t.Errorf("blank query has terms: %v", got)
	}
}
