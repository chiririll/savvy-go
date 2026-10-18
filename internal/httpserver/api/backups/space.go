package backups

import (
	"net/http"
	"os"
	backupx "savvy-go/internal/httpserver/httpx/backups"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
)

func (h Handler) spaceBackupsIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.Backups.SpaceBackups(httpx.Scope(r).ID)
	backupx.List(w, list, err)
}

func (h Handler) spaceBackupsStore(w http.ResponseWriter, r *http.Request) {
	space := httpx.CurrentSpace(w, r, h.Spaces)
	if space == nil {
		return
	}
	b, err := h.backupsService(r).CreateSpace(r.Context(), *space, backupx.ReadNote(r))
	if err != nil {
		backupx.WriteError(w, err)
		return
	}
	httpx.WriteData(w, http.StatusCreated, dto.NewBackup(*b))
}

// spaceUploadLimit bounds an upload to what the space may hold, with room for
// the archive itself.
func (h Handler) spaceUploadLimit(r *http.Request) int64 {
	if q := h.spaceQuota(r); q > 0 {
		return q + 16<<20
	}
	return backupx.MaxServerUpload
}

func (h Handler) spaceBackupsUpload(w http.ResponseWriter, r *http.Request) {
	tmp, name, ok := backupx.ReceiveUpload(w, r, h.BackupsDir, h.spaceUploadLimit(r))
	if !ok {
		return
	}
	if !backupx.RejectLaravel(w, tmp, name) {
		return
	}
	b, err := h.backupsService(r).IngestSpace(httpx.Scope(r).ID, tmp)
	if err != nil {
		_ = os.Remove(tmp)
		backupx.WriteError(w, err)
		return
	}
	httpx.WriteData(w, http.StatusCreated, dto.NewBackup(*b))
}

func (h Handler) spaceBackupsDownload(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if _, err := h.Backups.SpaceBackup(httpx.Scope(r).ID, name); err != nil {
		backupx.WriteError(w, err)
		return
	}
	backupx.Serve(w, r, h.Backups.SpacePath(httpx.Scope(r).ID, name), name)
}

func (h Handler) spaceBackupsRestore(w http.ResponseWriter, r *http.Request) {
	space := httpx.CurrentSpace(w, r, h.Spaces)
	if space == nil {
		return
	}
	if err := h.Backups.RestoreSpace(r.Context(), *space, chi.URLParam(r, "name"), h.spaceQuota(r)); err != nil {
		backupx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"message": "Backup restored."})
}

func (h Handler) spaceBackupsDestroy(w http.ResponseWriter, r *http.Request) {
	if err := h.Backups.DeleteSpace(httpx.Scope(r).ID, chi.URLParam(r, "name")); err != nil {
		backupx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
