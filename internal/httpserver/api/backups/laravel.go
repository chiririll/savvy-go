package backups

import (
	"net/http"
	"os"
	backupx "savvy-go/internal/httpserver/httpx/backups"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/legacy/restore"
)

// rejectLaravelBackup refuses a Laravel database where a backup archive is
// expected: it is upgraded by the dedicated endpoints, which save the result.
func (h Handler) rejectLaravelBackup(w http.ResponseWriter, tmp, name string) bool {
	if !restore.IsDatabaseFile(name) {
		return true
	}
	_ = os.Remove(tmp)
	httpx.WriteValidation(
		w,
		map[string][]string{"file": {"A Laravel database is not a backup: upgrade it from the upload dialog."}},
	)
	return false
}

// spaceBackupsUpgradeLaravel upgrades the finances of an uploaded Laravel
// database and saves only the upgraded version as a backup of the current
// space (space admins, by route).
func (h Handler) spaceBackupsUpgradeLaravel(w http.ResponseWriter, r *http.Request) {
	space := httpx.CurrentSpace(w, r, h.Spaces)
	if space == nil {
		return
	}
	tmp, _, ok := backupx.ReceiveUpload(w, r, h.BackupsDir, h.spaceUploadLimit(r))
	if !ok {
		return
	}
	defer os.Remove(tmp)
	b, err := restore.Space(r.Context(), h.backupsService(r), *space, tmp, h.spaceQuota(r))
	if err != nil {
		backupx.WriteError(w, err)
		return
	}
	httpx.WriteData(w, http.StatusCreated, dto.NewBackup(*b))
}
