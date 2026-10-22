// Package backups is what the server and the space backup routes share: error replies,
// listings, uploads and downloads.
package backups

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"

	"savvy-go/internal/backup"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/legacy/restore"
)

// MaxServerUpload bounds the upload of a server backup.
const MaxServerUpload = 4 << 30 // 4 GiB

// WriteError answers with a backup error: 404 for a missing backup, else 422.
func WriteError(w http.ResponseWriter, err error) {
	if errors.Is(err, backup.ErrBackupNotFound) {
		httpx.WriteMessage(w, http.StatusNotFound, "Backup not found.")
		return
	}
	httpx.WriteMessage(w, http.StatusUnprocessableEntity, err.Error())
}

// List answers with the backups in list.
func List(w http.ResponseWriter, list []backup.Entry, err error) {
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(list, dto.NewBackup))
}

// ReadNote reads the optional note of a backup request.
func ReadNote(r *http.Request) *string {
	var body struct {
		Note *string `json:"note"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	return body.Note
}

// ReceiveUpload stores the multipart "file" in a temporary file, at most
// limit bytes.
func ReceiveUpload(w http.ResponseWriter, r *http.Request, dir string, limit int64) (path, name string, ok bool) {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteValidation(w, map[string][]string{"file": {"The file field is required."}})
		return "", "", false
	}
	defer file.Close()
	if err := os.MkdirAll(dir, 0o775); err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return "", "", false
	}
	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return "", "", false
	}
	if _, err := io.Copy(tmp, file); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		httpx.WriteValidation(w, map[string][]string{"file": {"The file is too large."}})
		return "", "", false
	}
	_ = tmp.Close()
	return tmp.Name(), header.Filename, true
}

// Serve sends the backup file at path as a download named name.
func Serve(w http.ResponseWriter, r *http.Request, path, name string) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	http.ServeFile(w, r, path)
}

// RejectLaravel refuses a Laravel database where a backup archive is
// expected: it is upgraded by the dedicated endpoints, which save the result.
func RejectLaravel(w http.ResponseWriter, tmp, name string) bool {
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
