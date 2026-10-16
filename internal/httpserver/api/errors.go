package api

import (
	"errors"
	"net/http"

	"savvy-go/internal/ledger"
	"savvy-go/internal/refsync"
	"savvy-go/internal/store"
)

// WriteAccountError maps account create/update failures to responses.
func WriteAccountError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ledger.ErrAccountCurrencyImmutable):
		WriteMessage(w, 422, "Account currency cannot be changed after creation.")
	case errors.Is(err, ledger.ErrUnknownCurrency):
		WriteValidation(w, map[string][]string{"currency_id": {"The selected currency is invalid."}})
	default:
		WriteMessage(w, 422, err.Error())
	}
}

// WriteSyncError answers an error of a sync group. ok is false for errors it
// does not know, which the caller words itself.
func WriteSyncError(w http.ResponseWriter, err error) (ok bool) {
	var mirror *refsync.MirrorError
	switch {
	case errors.Is(err, refsync.ErrRights), errors.Is(err, refsync.ErrNotAdmin):
		WriteMessage(w, http.StatusForbidden, err.Error())
	case errors.Is(err, refsync.ErrNoGroup), errors.Is(err, store.ErrNotFound):
		WriteMessage(w, http.StatusNotFound, "Not found.")
	case errors.Is(err, store.ErrUnavailable):
		WriteMessage(w, http.StatusUnprocessableEntity, "A space of the group is temporarily unavailable.")
	case errors.As(err, &mirror), errors.Is(err, refsync.ErrUnresolved), errors.Is(err, refsync.ErrInvalid),
		errors.Is(err, refsync.ErrInGroup), errors.Is(err, refsync.ErrTooFew), errors.Is(err, refsync.ErrNotMember),
		errors.Is(err, refsync.ErrDiverged):
		WriteMessage(w, http.StatusUnprocessableEntity, err.Error())
	default:
		return false
	}
	return true
}
