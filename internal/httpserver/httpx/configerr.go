package httpx

import (
	"errors"
	"net/http"

	"savvy-go/internal/config"
)

// WriteConfigError answers a failed save of a config file setting, the same for every setting:
// 409 when the file cannot be written, 422 naming the key when the value is refused, else 500.
func WriteConfigError(w http.ResponseWriter, err error) {
	var field config.FieldError
	switch {
	case errors.Is(err, config.ErrReadOnly):
		WriteCoded(w, http.StatusConflict, "The config file is read-only: edit it by hand and restart.", "config_read_only")
	case errors.As(err, &field):
		WriteValidation(w, map[string][]string{field.Field: {field.Message}})
	default:
		WriteMessage(w, http.StatusInternalServerError, err.Error())
	}
}
