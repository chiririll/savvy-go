package backups

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/dto"
)

func (h Handler) spaceBackupsIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.Backups.SpaceBackups(api.Scope(r).ID)
	List(w, list, err)
}

func (h Handler) spaceBackupsStore(w http.ResponseWriter, r *http.Request) {
	space := api.CurrentSpace(w, r, h.Spaces)
	if space == nil {
		return
	}
	b, err := h.backupsService(r).CreateSpace(r.Context(), *space, readNote(r))
	if err != nil {
		WriteError(w, err)
		return
	}
	api.WriteData(w, http.StatusCreated, dto.NewBackup(*b))
}

// spaceUploadLimit bounds an upload to what the space may hold, with room for
// the archive itself.
func (h Handler) spaceUploadLimit(r *http.Request) int64 {
	if q := h.spaceQuota(r); q > 0 {
		return q + 16<<20
	}
	return maxServerUpload
}

func (h Handler) spaceBackupsUpload(w http.ResponseWriter, r *http.Request) {
	tmp, name, ok := h.receiveUpload(w, r, h.spaceUploadLimit(r))
	if !ok {
		return
	}
	if !h.rejectLaravelBackup(w, tmp, name) {
		return
	}
	b, err := h.backupsService(r).IngestSpace(api.Scope(r).ID, tmp)
	if err != nil {
		_ = os.Remove(tmp)
		WriteError(w, err)
		return
	}
	api.WriteData(w, http.StatusCreated, dto.NewBackup(*b))
}

func (h Handler) spaceBackupsDownload(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if _, err := h.Backups.SpaceBackup(api.Scope(r).ID, name); err != nil {
		WriteError(w, err)
		return
	}
	serveBackup(w, r, h.Backups.SpacePath(api.Scope(r).ID, name), name)
}

func (h Handler) spaceBackupsRestore(w http.ResponseWriter, r *http.Request) {
	space := api.CurrentSpace(w, r, h.Spaces)
	if space == nil {
		return
	}
	if err := h.Backups.RestoreSpace(r.Context(), *space, chi.URLParam(r, "name"), h.spaceQuota(r)); err != nil {
		WriteError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]string{"message": "Backup restored."})
}

func (h Handler) spaceBackupsDestroy(w http.ResponseWriter, r *http.Request) {
	if err := h.Backups.DeleteSpace(api.Scope(r).ID, chi.URLParam(r, "name")); err != nil {
		WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
