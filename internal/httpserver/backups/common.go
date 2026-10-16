package backups

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"

	"savvy-go/internal/backup"
	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/dto"
)

// WriteError answers with a backup error: 404 for a missing backup, else 422.
func WriteError(w http.ResponseWriter, err error) {
	if errors.Is(err, backup.ErrBackupNotFound) {
		api.WriteMessage(w, http.StatusNotFound, "Backup not found.")
		return
	}
	api.WriteMessage(w, http.StatusUnprocessableEntity, err.Error())
}

// List answers with the backups in list.
func List(w http.ResponseWriter, list []backup.Entry, err error) {
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	api.WriteData(w, http.StatusOK, dto.Map(list, dto.NewBackup))
}

func readNote(r *http.Request) *string {
	var body struct {
		Note *string `json:"note"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	return body.Note
}

// receiveUpload stores the multipart "file" in a temporary file, at most
// limit bytes.
func (h Handler) receiveUpload(w http.ResponseWriter, r *http.Request, limit int64) (path, name string, ok bool) {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	file, header, err := r.FormFile("file")
	if err != nil {
		api.WriteValidation(w, map[string][]string{"file": {"The file field is required."}})
		return "", "", false
	}
	defer file.Close()
	if err := os.MkdirAll(h.Cfg.BackupsDir, 0o775); err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return "", "", false
	}
	tmp, err := os.CreateTemp(h.Cfg.BackupsDir, ".upload-*")
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return "", "", false
	}
	if _, err := io.Copy(tmp, file); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		api.WriteValidation(w, map[string][]string{"file": {"The file is too large."}})
		return "", "", false
	}
	_ = tmp.Close()
	return tmp.Name(), header.Filename, true
}

func serveBackup(w http.ResponseWriter, r *http.Request, path, name string) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	http.ServeFile(w, r, path)
}
