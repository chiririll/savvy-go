// Package backups serves the backups of the whole server: its databases and signing
// key. Server admins only.
package backups

import (
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/backup"
	"savvy-go/internal/httpserver/httpx"
	backupx "savvy-go/internal/httpserver/httpx/backups"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/legacy/restore"
)

// Handler holds what the server backup routes need.
type Handler struct {
	BackupsDir string
	Backups    backup.Service
}

func (h Handler) backupsIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.Backups.ServerBackups()
	backupx.List(w, list, err)
}

func (h Handler) backupsStore(w http.ResponseWriter, r *http.Request) {
	b, err := h.Backups.CreateServer(r.Context(), backupx.ReadNote(r))
	if err != nil {
		backupx.WriteError(w, err)
		return
	}
	httpx.WriteData(w, http.StatusCreated, dto.NewBackup(*b))
}

func (h Handler) backupsUpload(w http.ResponseWriter, r *http.Request) {
	tmp, name, ok := backupx.ReceiveUpload(w, r, h.BackupsDir, backupx.MaxServerUpload)
	if !ok {
		return
	}
	if !backupx.RejectLaravel(w, tmp, name) {
		return
	}
	b, err := h.Backups.IngestServer(tmp)
	if err != nil {
		_ = os.Remove(tmp)
		backupx.WriteError(w, err)
		return
	}
	httpx.WriteData(w, http.StatusCreated, dto.NewBackup(*b))
}

func (h Handler) backupsDownload(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if _, err := h.Backups.ServerBackup(name); err != nil {
		backupx.WriteError(w, err)
		return
	}
	backupx.Serve(w, r, h.Backups.ServerPath(name), name)
}

func (h Handler) backupsRestore(w http.ResponseWriter, r *http.Request) {
	if err := h.Backups.RestoreServer(r.Context(), chi.URLParam(r, "name")); err != nil {
		backupx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"message": "Backup restored."})
}

func (h Handler) backupsDestroy(w http.ResponseWriter, r *http.Request) {
	if err := h.Backups.DeleteServer(chi.URLParam(r, "name")); err != nil {
		backupx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// backupsUpgradeLaravel upgrades an uploaded Laravel database and saves only
// the upgraded version as a server backup. Nothing live changes until the
// admin restores it (server admins, by route).
func (h Handler) backupsUpgradeLaravel(w http.ResponseWriter, r *http.Request) {
	tmp, _, ok := backupx.ReceiveUpload(w, r, h.BackupsDir, backupx.MaxServerUpload)
	if !ok {
		return
	}
	defer os.Remove(tmp)
	// An invalid key is refused by the store before anything is converted.
	b, err := restore.Server(r.Context(), h.Backups, tmp, strings.TrimSpace(r.FormValue("app_key")))
	if err != nil {
		backupx.WriteError(w, err)
		return
	}
	httpx.WriteData(w, http.StatusCreated, dto.NewBackup(*b))
}
