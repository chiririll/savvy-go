package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/chiririll/savvy-plus/internal/domain"
	"github.com/go-chi/chi/v5"
)

func (s *Server) backupsIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.backups.All(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, mapSlice(list, domain.Backup.JSON))
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
	writeData(w, http.StatusCreated, b.JSON())
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
	writeJSON(w, http.StatusOK, b.JSON())
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

func (s *Server) backupsInspect(w http.ResponseWriter, r *http.Request) {
	b := s.backupParam(w, r)
	if b == nil {
		return
	}
	out, err := s.backups.Inspect(r.Context(), *b)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) backupsRestore(w http.ResponseWriter, r *http.Request) {
	b := s.backupParam(w, r)
	if b == nil {
		return
	}
	if err := s.backups.Restore(r.Context(), *b); err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
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
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	b, _ := s.backups.ByID(r.Context(), id)
	if b == nil {
		writeMessage(w, http.StatusNotFound, "Not found.")
	}
	return b
}
