package sqlite

import (
	"errors"
	"os"

	sqlite3 "modernc.org/sqlite"

	"savvy-go/internal/store"
)

func removeFiles(path string) {
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		_ = os.Remove(path + suffix)
	}
}

// quotaError keeps the driver error for the log while matching
// store.ErrQuotaExceeded.
type quotaError struct{ err error }

func (e quotaError) Error() string { return store.ErrQuotaExceeded.Error() + ": " + e.err.Error() }

func (e quotaError) Unwrap() []error { return []error{store.ErrQuotaExceeded, e.err} }

// constraintError matches store.ErrConstraint and says nothing of the schema;
// the driver error stays reachable by unwrapping, for the log.
type constraintError struct{ err error }

func (e constraintError) Error() string { return store.ErrConstraint.Error() }

func (e constraintError) Unwrap() []error { return []error{store.ErrConstraint, e.err} }

// mapErr turns SQLITE_FULL (the file reached max_page_count) into
// store.ErrQuotaExceeded and a broken constraint into store.ErrConstraint.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	var se *sqlite3.Error
	if errors.As(err, &se) && se.Code()&0xff == sqliteFull {
		return quotaError{err}
	}
	if errors.As(err, &se) && se.Code()&0xff == sqliteConstraint {
		return constraintError{err}
	}
	return err
}
