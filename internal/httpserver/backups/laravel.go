package backups

import (
	"net/http"
	"os"
	"strings"

	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/dto"
	"savvy-go/internal/legacy/restore"
)

// rejectLaravelBackup refuses a Laravel database where a backup archive is
// expected: it is upgraded by the dedicated endpoints, which save the result.
func (h Handler) rejectLaravelBackup(w http.ResponseWriter, tmp, name string) bool {
	if !restore.IsDatabaseFile(name) {
		return true
	}
	_ = os.Remove(tmp)
	api.WriteValidation(w, map[string][]string{"file": {"A Laravel database is not a backup: upgrade it from the upload dialog."}})
	return false
}

// backupsUpgradeLaravel upgrades an uploaded Laravel database and saves only
// the upgraded version as a server backup. Nothing live changes until the
// admin restores it (server admins, by route).
func (h Handler) backupsUpgradeLaravel(w http.ResponseWriter, r *http.Request) {
	tmp, _, ok := h.receiveUpload(w, r, maxServerUpload)
	if !ok {
		return
	}
	defer os.Remove(tmp)
	// An invalid key is refused by the store before anything is converted.
	b, err := restore.Server(r.Context(), h.Backups, tmp, strings.TrimSpace(r.FormValue("app_key")))
	if err != nil {
		WriteError(w, err)
		return
	}
	api.WriteData(w, http.StatusCreated, dto.NewBackup(*b))
}

// spaceBackupsUpgradeLaravel upgrades the finances of an uploaded Laravel
// database and saves only the upgraded version as a backup of the current
// space (space admins, by route).
func (h Handler) spaceBackupsUpgradeLaravel(w http.ResponseWriter, r *http.Request) {
	space := api.CurrentSpace(w, r, h.Spaces)
	if space == nil {
		return
	}
	tmp, _, ok := h.receiveUpload(w, r, h.spaceUploadLimit(r))
	if !ok {
		return
	}
	defer os.Remove(tmp)
	b, err := restore.Space(r.Context(), h.backupsService(r), *space, tmp, h.spaceQuota(r))
	if err != nil {
		WriteError(w, err)
		return
	}
	api.WriteData(w, http.StatusCreated, dto.NewBackup(*b))
}
