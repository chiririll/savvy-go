package backups

import (
	"net/http"
	"os"
	"path/filepath"

	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/legacy/restore"
	"savvy-go/internal/spaces"
)

// spacesImport creates a new space, owned by the caller, from an uploaded
// space backup (or a Laravel database). Guests cannot.
func (h Handler) spacesImport(w http.ResponseWriter, r *http.Request) {
	u := api.User(r)
	if u.IsGuest() {
		api.WriteMessage(w, http.StatusForbidden, "Guests cannot create spaces.")
		return
	}
	tmp, name, ok := h.receiveUpload(w, r, h.spaceUploadLimit(r))
	if !ok {
		return
	}
	defer os.Remove(tmp)
	// Keep the extension: it tells an archive from a bare database file.
	src := tmp + filepath.Ext(name)
	if err := os.Rename(tmp, src); err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
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
		WriteError(w, err)
		return
	}
	api.WriteData(w, http.StatusCreated, map[string]any{"id": space.ID, "uuid": space.UUID, "name": space.Name, "role": space.Role})
}
