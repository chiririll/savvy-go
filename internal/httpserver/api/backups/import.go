package backups

import (
	"net/http"
	"os"
	"path/filepath"
	backupx "savvy-go/internal/httpserver/httpx/backups"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/legacy/restore"
	"savvy-go/internal/spaces"
)

// spacesImport creates a new space, owned by the caller, from an uploaded
// space backup (or a Laravel database). Guests cannot.
func (h Handler) spacesImport(w http.ResponseWriter, r *http.Request) {
	u := httpx.User(r)
	if u.IsGuest() {
		httpx.WriteMessage(w, http.StatusForbidden, "Guests cannot create spaces.")
		return
	}
	tmp, name, ok := backupx.ReceiveUpload(w, r, h.BackupsDir, h.spaceUploadLimit(r))
	if !ok {
		return
	}
	defer os.Remove(tmp)
	// Keep the extension: it tells an archive from a bare database file.
	src := tmp + filepath.Ext(name)
	if err := os.Rename(tmp, src); err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer os.Remove(src)
	var space *spaces.Space
	var err error
	if restore.IsDatabaseFile(src) {
		space, err = restore.ImportSpace(r.Context(), h.Spaces, src, r.FormValue("name"), u, h.spaceQuota(r))
	} else {
		space, err = h.Backups.ImportSpace(r.Context(), h.Spaces, src, r.FormValue("name"), u, h.spaceQuota(r))
	}
	if err != nil {
		backupx.WriteError(w, err)
		return
	}
	httpx.WriteData(w, http.StatusCreated, dto.NewSpace(*space))
}
