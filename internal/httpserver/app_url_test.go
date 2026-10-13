package httpserver

import "testing"

func TestCleanAppURL(t *testing.T) {
	ok := map[string]string{
		"":                           "",
		"  ":                         "",
		"https://savvy.example.com/": "https://savvy.example.com",
		"http://localhost:3000":      "http://localhost:3000",
	}
	for in, want := range ok {
		got, err := cleanAppURL(in)
		if err != nil || got != want {
			t.Errorf("cleanAppURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []any{"savvy.example.com", "ftp://x", "https://", "https://x/?a=1", 5} {
		if _, err := cleanAppURL(in); err == nil {
			t.Errorf("cleanAppURL(%v) accepted", in)
		}
	}
}
