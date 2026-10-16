package backups

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/dto"
)

func (h Handler) backupsIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.Backups.ServerBackups()
	List(w, list, err)
}

func (h Handler) backupsStore(w http.ResponseWriter, r *http.Request) {
	b, err := h.Backups.CreateServer(r.Context(), readNote(r))
	if err != nil {
		WriteError(w, err)
		return
	}
	api.WriteData(w, http.StatusCreated, dto.NewBackup(*b))
}

func (h Handler) backupsUpload(w http.ResponseWriter, r *http.Request) {
	tmp, name, ok := h.receiveUpload(w, r, maxServerUpload)
	if !ok {
		return
	}
	if !h.rejectLaravelBackup(w, tmp, name) {
		return
	}
	b, err := h.Backups.IngestServer(tmp)
	if err != nil {
		_ = os.Remove(tmp)
		WriteError(w, err)
		return
	}
	api.WriteData(w, http.StatusCreated, dto.NewBackup(*b))
}

func (h Handler) backupsDownload(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if _, err := h.Backups.ServerBackup(name); err != nil {
		WriteError(w, err)
		return
	}
	serveBackup(w, r, h.Backups.ServerPath(name), name)
}

func (h Handler) backupsRestore(w http.ResponseWriter, r *http.Request) {
	if err := h.Backups.RestoreServer(r.Context(), chi.URLParam(r, "name")); err != nil {
		WriteError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]string{"message": "Backup restored."})
}

func (h Handler) backupsDestroy(w http.ResponseWriter, r *http.Request) {
	if err := h.Backups.DeleteServer(chi.URLParam(r, "name")); err != nil {
		WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
