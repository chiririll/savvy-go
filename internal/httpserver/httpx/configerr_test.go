package httpx

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"savvy-go/internal/config"
)

func TestWriteConfigErrorMapsSaveFailures(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{"read-only", config.ErrReadOnly, http.StatusConflict, `"error":"config_read_only"`},
		{"field", config.FieldError{Field: "host", Message: "mailer.host is empty"}, http.StatusUnprocessableEntity, `"host":["mailer.host is empty"]`},
		{"wrapped field", fmt.Errorf("save: %w", config.FieldError{Field: "port", Message: "bad port"}), http.StatusUnprocessableEntity, `"port":["bad port"]`},
		{"other", errors.New("disk full"), http.StatusInternalServerError, "disk full"},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		WriteConfigError(w, c.err)
		if w.Code != c.status || !strings.Contains(w.Body.String(), c.body) {
			t.Errorf("%s: %d %s; want %d with %s", c.name, w.Code, w.Body.String(), c.status, c.body)
		}
	}
}
