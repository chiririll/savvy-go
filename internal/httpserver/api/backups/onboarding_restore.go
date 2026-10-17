package backups

import (
	_ "embed"
	"errors"
	"net/http"
	"os"
	"savvy-go/internal/httpserver/httpx"
	backupx "savvy-go/internal/httpserver/httpx/backups"
	"strings"

	"savvy-go/internal/firstrun"
	"savvy-go/internal/legacy/restore"
)

var errBackupWithoutUsers = errors.New("restore: the backup has no users")

// onboardingRestore is the usual server restore for a server that has no
// administrator yet: the bootstrap password authorizes it instead of a session.
// The backup's users, settings, signing key and spaces replace the empty
// server, which also drops every session, so nobody is logged in afterwards.
// A failed restore leaves the password valid.
func (h Handler) onboardingRestore(w http.ResponseWriter, r *http.Request) {
	if !h.restoreOpen(w, r) {
		return
	}
	// The upload can be gigabytes: refuse a wrong password before reading it.
	// Register checks again, under its lock.
	r.Body = http.MaxBytesReader(w, r.Body, backupx.MaxServerUpload)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		httpx.WriteValidation(w, map[string][]string{"file": {"The file field is required."}})
		return
	}
	defer r.MultipartForm.RemoveAll()
	password := r.FormValue("bootstrap_password")
	if h.Bootstrap.Verify(password) != nil {
		writeWrongPassword(w)
		return
	}
	tmp, name, ok := backupx.ReceiveUpload(w, r, h.BackupsDir, backupx.MaxServerUpload)
	if !ok {
		return
	}
	defer os.Remove(tmp)

	// A restore that fails has not spent the password; Register keeps it valid.
	var saved string
	err := h.Bootstrap.Register(password, func() error {
		var err error
		saved, err = h.restoreFirstRun(r, tmp, name)
		if err != nil {
			return err
		}
		return h.requireUsers(r)
	})
	if err != nil && saved != "" {
		// The backup was only needed for this restore.
		_ = h.Backups.DeleteServer(saved)
	}
	writeRestoreResult(w, err)
}

// restoreOpen reports whether restoring is allowed, and answers if it is not.
func (h Handler) restoreOpen(w http.ResponseWriter, r *http.Request) bool {
	n, err := h.Users.Count(r.Context())
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return false
	}
	if n > 0 || !h.Bootstrap.Active() {
		httpx.WriteMessage(w, http.StatusForbidden, "Restoring is closed.")
		return false
	}
	return true
}

// requireUsers fails when the restored server has no users: nobody could log
// in to it.
func (h Handler) requireUsers(r *http.Request) error {
	n, err := h.Users.Count(r.Context())
	if err != nil {
		return err
	}
	if n == 0 {
		return errBackupWithoutUsers
	}
	return nil
}

func writeWrongPassword(w http.ResponseWriter) {
	httpx.WriteValidation(w, map[string][]string{
		"bootstrap_password": {"The bootstrap password is missing or wrong."},
	})
}

func writeRestoreResult(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, firstrun.ErrWrongPassword):
		writeWrongPassword(w)
	case errors.Is(err, firstrun.ErrClosed):
		httpx.WriteMessage(w, http.StatusForbidden, "Restoring is closed.")
	case errors.Is(err, errBackupWithoutUsers):
		httpx.WriteValidation(w, map[string][]string{"file": {"The backup has no users."}})
	case err != nil:
		backupx.WriteError(w, err)
	default:
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"message": "Backup restored."})
	}
}

// restoreFirstRun saves the upload as a server backup (a Laravel database is
// upgraded first) and restores it. It returns the backup's name.
func (h Handler) restoreFirstRun(r *http.Request, tmp, name string) (string, error) {
	if restore.IsDatabaseFile(name) {
		b, err := restore.Server(r.Context(), h.Backups, tmp, strings.TrimSpace(r.FormValue("app_key")))
		if err != nil {
			return "", err
		}
		return b.Filename, h.Backups.RestoreServer(r.Context(), b.Filename)
	}
	b, err := h.Backups.IngestServer(tmp)
	if err != nil {
		return "", err
	}
	return b.Filename, h.Backups.RestoreServer(r.Context(), b.Filename)
}
