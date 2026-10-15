package httpserver

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"net/http"
	"os"
	"strings"

	"savvy-go/internal/bootstrap"
	"savvy-go/internal/legacy/restore"
)

var errBackupWithoutUsers = errors.New("restore: the backup has no users")

// onboardingRestore is the usual server restore for a server that has no
// administrator yet: the bootstrap password authorizes it instead of a session.
// The backup's users, settings, signing key and spaces replace the empty
// server, which also drops every session, so nobody is logged in afterwards.
// A failed restore leaves the password valid.
func (s *Server) onboardingRestore(w http.ResponseWriter, r *http.Request) {
	n, err := s.users.Count(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n > 0 || !s.bootstrap.Active() {
		writeMessage(w, http.StatusForbidden, "Restoring is closed.")
		return
	}
	// The upload can be gigabytes: refuse a wrong password before reading it.
	// Register checks again, under its lock.
	r.Body = http.MaxBytesReader(w, r.Body, maxServerUpload)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeValidation(w, map[string][]string{"file": {"The file field is required."}})
		return
	}
	defer r.MultipartForm.RemoveAll()
	password := r.FormValue("bootstrap_password")
	if s.bootstrap.Verify(password) != nil {
		writeValidation(w, map[string][]string{"bootstrap_password": {"The bootstrap password is missing or wrong."}})
		return
	}
	tmp, name, ok := s.receiveUpload(w, r, maxServerUpload)
	if !ok {
		return
	}
	defer os.Remove(tmp)

	// A restore that fails has not spent the password; Register keeps it valid.
	var saved string
	err = s.bootstrap.Register(password, func() error {
		var err error
		if saved, err = s.restoreFirstRun(r, tmp, name); err != nil {
			return err
		}
		// A backup without users would leave a server nobody can log in to.
		if n, err := s.users.Count(r.Context()); err != nil {
			return err
		} else if n == 0 {
			return errBackupWithoutUsers
		}
		return nil
	})
	if err != nil && saved != "" {
		// The backup was only needed for this restore.
		_ = s.backups.DeleteServer(saved)
	}
	switch {
	case errors.Is(err, bootstrap.ErrWrongPassword):
		writeValidation(w, map[string][]string{"bootstrap_password": {"The bootstrap password is missing or wrong."}})
	case errors.Is(err, bootstrap.ErrClosed):
		writeMessage(w, http.StatusForbidden, "Restoring is closed.")
	case errors.Is(err, errBackupWithoutUsers):
		writeValidation(w, map[string][]string{"file": {"The backup has no users."}})
	case err != nil:
		writeBackupError(w, err)
	default:
		writeJSON(w, http.StatusOK, map[string]string{"message": "Backup restored."})
	}
}

// restoreFirstRun saves the upload as a server backup (a Laravel database is
// upgraded first) and restores it. It returns the backup's name.
func (s *Server) restoreFirstRun(r *http.Request, tmp, name string) (string, error) {
	if restore.IsDatabaseFile(name) {
		b, err := restore.Server(r.Context(), s.backups, tmp, strings.TrimSpace(r.FormValue("app_key")))
		if err != nil {
			return "", err
		}
		return b.Filename, s.backups.RestoreServer(r.Context(), b.Filename)
	}
	b, err := s.backups.IngestServer(tmp)
	if err != nil {
		return "", err
	}
	return b.Filename, s.backups.RestoreServer(r.Context(), b.Filename)
}

func (s *Server) publicOnboardingRestoreRoutes(r chi.Router) {
	r.Post("/onboarding/restore", s.onboardingRestore)
}
