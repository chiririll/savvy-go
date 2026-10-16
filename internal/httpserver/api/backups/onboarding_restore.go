package backups

import (
	_ "embed"
	"errors"
	"net/http"
	"os"
	"savvy-go/internal/httpserver/httpx"
	backupx "savvy-go/internal/httpserver/httpx/backups"
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
func (h Handler) onboardingRestore(w http.ResponseWriter, r *http.Request) {
	n, err := h.Users.Count(r.Context())
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n > 0 || !h.Bootstrap.Active() {
		httpx.WriteMessage(w, http.StatusForbidden, "Restoring is closed.")
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
		httpx.WriteValidation(w, map[string][]string{"bootstrap_password": {"The bootstrap password is missing or wrong."}})
		return
	}
	tmp, name, ok := backupx.ReceiveUpload(w, r, h.Cfg.BackupsDir, backupx.MaxServerUpload)
	if !ok {
		return
	}
	defer os.Remove(tmp)

	// A restore that fails has not spent the password; Register keeps it valid.
	var saved string
	err = h.Bootstrap.Register(password, func() error {
		var err error
		if saved, err = h.restoreFirstRun(r, tmp, name); err != nil {
			return err
		}
		// A backup without users would leave a server nobody can log in to.
		if n, err := h.Users.Count(r.Context()); err != nil {
			return err
		} else if n == 0 {
			return errBackupWithoutUsers
		}
		return nil
	})
	if err != nil && saved != "" {
		// The backup was only needed for this restore.
		_ = h.Backups.DeleteServer(saved)
	}
	switch {
	case errors.Is(err, bootstrap.ErrWrongPassword):
		httpx.WriteValidation(w, map[string][]string{"bootstrap_password": {"The bootstrap password is missing or wrong."}})
	case errors.Is(err, bootstrap.ErrClosed):
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
