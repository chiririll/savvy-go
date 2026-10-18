package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"savvy-go/internal/httpserver/httpx"
)

func TestReplyBodies(t *testing.T) {
	cases := []struct {
		name  string
		write func(w http.ResponseWriter)
		want  string
	}{
		{"message", func(w http.ResponseWriter) { httpx.WriteMessage(w, 400, "no") }, `{"message":"no"}`},
		{"coded", func(w http.ResponseWriter) { httpx.WriteCoded(w, 422, "no", "why") }, `{"message":"no","error":"why"}`},
		{"success", httpx.WriteSuccess, `{"success":true}`},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		c.write(rec)
		if got := strings.TrimSpace(rec.Body.String()); got != c.want {
			t.Errorf("%s: body %s, want %s", c.name, got, c.want)
		}
	}
}
