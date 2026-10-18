package httpx

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// Lookup loads the resource named by the {id} path segment. It answers 404
// for an id that is not a number or matches nothing, 500 when the lookup
// itself fails, and returns nil in both cases.
func Lookup[T any](w http.ResponseWriter, r *http.Request, byID func(context.Context, int64) (*T, error)) *T {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		WriteMessage(w, http.StatusNotFound, "Not found.")
		return nil
	}
	v, err := byID(r.Context(), id)
	switch {
	case err != nil:
		WriteMessage(w, http.StatusInternalServerError, err.Error())
		return nil
	case v == nil:
		WriteMessage(w, http.StatusNotFound, "Not found.")
		return nil
	}
	return v
}
