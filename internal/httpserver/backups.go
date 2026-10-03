package httpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"savvy-go/internal/domain"
	"savvy-go/internal/httpserver/dto"
	"savvy-go/internal/legacy"

	"github.com/go-chi/chi/v5"
)

func (s *Server) backupsIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.backups.All(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]dto.Backup, 0, len(list))
	for _, b := range list {
		out = append(out, s.backupView(r.Context(), b))
	}
	writeData(w, http.StatusOK, out)
}

func (s *Server) backupsStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Note *string `json:"note"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	b, err := s.backups.Create(r.Context(), body.Note)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusCreated, s.backupView(r.Context(), *b))
}

func (s *Server) backupsUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		writeValidation(w, map[string][]string{"file": {"The file field is required."}})
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeValidation(w, map[string][]string{"file": {"The file field is required."}})
		return
	}
	defer file.Close()
	tmp, err := os.CreateTemp(s.cfg.BackupsDir, "upload-*.sqlite")
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	_, _ = io.Copy(tmp, file)
	tmp.Close()
	note := r.FormValue("note")
	var notePtr *string
	if note != "" {
		notePtr = &note
	}
	// Treat uploaded file as the next backup by copying through Create after replacing db?
	// Store the uploaded sqlite as a backup file directly.
	b, err := s.backups.Ingest(r.Context(), tmp.Name(), notePtr)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s.backupView(r.Context(), *b))
}

func (s *Server) backupsDownload(w http.ResponseWriter, r *http.Request) {
	b := s.backupParam(w, r)
	if b == nil {
		return
	}
	path := s.backups.Path(*b)
	w.Header().Set("Content-Type", "application/x-sqlite3")
	w.Header().Set("Content-Disposition", `attachment; filename="`+b.Filename+`"`)
	http.ServeFile(w, r, path)
}

// backupView describes b together with what restoring it here would do.
func (s *Server) backupView(ctx context.Context, b domain.Backup) dto.Backup {
	status, pending := s.backupStatus(ctx, b)
	return dto.NewBackup(b, status, pending)
}

// backupStatus classifies b for restore and counts the migrations restore
// would apply. It reads the file, so it matches what restore will do.
func (s *Server) backupStatus(ctx context.Context, b domain.Backup) (string, int) {
	info, err := legacy.InspectFile(ctx, s.backups.Path(b))
	if err != nil {
		return dto.BackupInvalid, 0
	}
	if info.Laravel {
		if !info.Supported {
			return dto.BackupLegacyUnsupported, 0
		}
		return dto.BackupLegacy, 0
	}
	pending, unknown, err := s.backups.Migrations(ctx, b)
	switch {
	case err != nil:
		return dto.BackupInvalid, 0
	case len(unknown) > 0:
		return dto.BackupNewer, 0
	case len(pending) > 0:
		return dto.BackupOutdated, len(pending)
	}
	return dto.BackupCurrent, 0
}

// upgradeBackup brings a staged backup copy up to the current schema before it
// replaces the live database.
func (s *Server) upgradeBackup(ctx context.Context, staged *sql.DB) error {
	return legacy.Upgrade(ctx, staged, s.cfg.AppKey)
}

func (s *Server) backupsRestore(w http.ResponseWriter, r *http.Request) {
	b := s.backupParam(w, r)
	if b == nil {
		return
	}
	if status, _ := s.backupStatus(r.Context(), *b); !dto.BackupRestorable(status) {
		writeMessage(w, 422, "This backup cannot be restored by this version of the app.")
		return
	}
	newDB, err := s.backups.Restore(r.Context(), *b, s.upgradeBackup)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	s.reconnect(newDB)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Backup restored."})
}

func (s *Server) backupsDestroy(w http.ResponseWriter, r *http.Request) {
	b := s.backupParam(w, r)
	if b == nil {
		return
	}
	if err := s.backups.Delete(r.Context(), *b); err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) backupParam(w http.ResponseWriter, r *http.Request) *domain.Backup {
	b, _ := s.backups.ByName(r.Context(), chi.URLParam(r, "name"))
	if b == nil {
		writeMessage(w, http.StatusNotFound, "Not found.")
	}
	return b
}
