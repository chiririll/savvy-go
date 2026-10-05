package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/domain"
	"savvy-go/internal/httpserver/dto"
	"savvy-go/internal/settings"
)

// Server backups hold every database and the signing key: server admins only
// (requireAdmin on the routes). Space backups hold one space: its admins.

const maxServerUpload = 4 << 30 // 4 GiB

func (s *Server) backupsService(r *http.Request) domain.Backups {
	b := s.backups
	b.KeepPerSpace = int(s.settings.Int(r.Context(), "space_backups_max", 10))
	return b
}

// spaceQuota is the size limit of the request's space, or the default limit
// for a new space outside one.
func (s *Server) spaceQuota(r *http.Request) int64 {
	if scope := sp(r); scope != nil {
		return s.spaces.Quota(r.Context(), scope.id)
	}
	return settings.SpaceQuota(r.Context(), s.store.Server())
}

func writeBackupError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrBackupNotFound) {
		writeMessage(w, http.StatusNotFound, "Backup not found.")
		return
	}
	writeMessage(w, http.StatusUnprocessableEntity, err.Error())
}

func backupList(w http.ResponseWriter, list []domain.Backup, err error) {
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, dto.Map(list, dto.NewBackup))
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
func (s *Server) receiveUpload(w http.ResponseWriter, r *http.Request, limit int64) (path, name string, ok bool) {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	file, header, err := r.FormFile("file")
	if err != nil {
		writeValidation(w, map[string][]string{"file": {"The file field is required."}})
		return "", "", false
	}
	defer file.Close()
	if err := os.MkdirAll(s.cfg.BackupsDir, 0o775); err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return "", "", false
	}
	tmp, err := os.CreateTemp(s.cfg.BackupsDir, ".upload-*")
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return "", "", false
	}
	if _, err := io.Copy(tmp, file); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		writeValidation(w, map[string][]string{"file": {"The file is too large."}})
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

// --- server ----------------------------------------------------------------

func (s *Server) backupsIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.backups.ServerBackups()
	backupList(w, list, err)
}

func (s *Server) backupsStore(w http.ResponseWriter, r *http.Request) {
	b, err := s.backups.CreateServer(r.Context(), readNote(r))
	if err != nil {
		writeBackupError(w, err)
		return
	}
	writeData(w, http.StatusCreated, dto.NewBackup(*b))
}

func (s *Server) backupsUpload(w http.ResponseWriter, r *http.Request) {
	tmp, name, ok := s.receiveUpload(w, r, maxServerUpload)
	if !ok {
		return
	}
	b, err := s.backups.IngestServer(tmp, name)
	if err != nil {
		_ = os.Remove(tmp)
		writeBackupError(w, err)
		return
	}
	writeData(w, http.StatusCreated, dto.NewBackup(*b))
}

func (s *Server) backupsDownload(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if _, err := s.backups.ServerBackup(name); err != nil {
		writeBackupError(w, err)
		return
	}
	serveBackup(w, r, s.backups.ServerPath(name), name)
}

func (s *Server) backupsRestore(w http.ResponseWriter, r *http.Request) {
	if err := s.backups.RestoreServer(r.Context(), chi.URLParam(r, "name")); err != nil {
		writeBackupError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Backup restored."})
}

func (s *Server) backupsDestroy(w http.ResponseWriter, r *http.Request) {
	if err := s.backups.DeleteServer(chi.URLParam(r, "name")); err != nil {
		writeBackupError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- space -----------------------------------------------------------------

// currentSpace is the registry entry of the request's space.
func (s *Server) currentSpace(w http.ResponseWriter, r *http.Request) *domain.Space {
	space, err := s.spaces.Get(r.Context(), sp(r).id)
	if err != nil || space == nil {
		writeMessage(w, http.StatusNotFound, "Space not found.")
		return nil
	}
	return space
}

func (s *Server) spaceBackupsIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.backups.SpaceBackups(sp(r).id)
	backupList(w, list, err)
}

func (s *Server) spaceBackupsStore(w http.ResponseWriter, r *http.Request) {
	space := s.currentSpace(w, r)
	if space == nil {
		return
	}
	b, err := s.backupsService(r).CreateSpace(r.Context(), *space, readNote(r))
	if err != nil {
		writeBackupError(w, err)
		return
	}
	writeData(w, http.StatusCreated, dto.NewBackup(*b))
}

// spaceUploadLimit bounds an upload to what the space may hold, with room for
// the archive itself.
func (s *Server) spaceUploadLimit(r *http.Request) int64 {
	if q := s.spaceQuota(r); q > 0 {
		return q + 16<<20
	}
	return maxServerUpload
}

func (s *Server) spaceBackupsUpload(w http.ResponseWriter, r *http.Request) {
	tmp, name, ok := s.receiveUpload(w, r, s.spaceUploadLimit(r))
	if !ok {
		return
	}
	b, err := s.backupsService(r).IngestSpace(sp(r).id, tmp, name)
	if err != nil {
		_ = os.Remove(tmp)
		writeBackupError(w, err)
		return
	}
	writeData(w, http.StatusCreated, dto.NewBackup(*b))
}

func (s *Server) spaceBackupsDownload(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if _, err := s.backups.SpaceBackup(sp(r).id, name); err != nil {
		writeBackupError(w, err)
		return
	}
	serveBackup(w, r, s.backups.SpacePath(sp(r).id, name), name)
}

func (s *Server) spaceBackupsRestore(w http.ResponseWriter, r *http.Request) {
	space := s.currentSpace(w, r)
	if space == nil {
		return
	}
	if err := s.backups.RestoreSpace(r.Context(), *space, chi.URLParam(r, "name"), s.spaceQuota(r)); err != nil {
		writeBackupError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Backup restored."})
}

func (s *Server) spaceBackupsDestroy(w http.ResponseWriter, r *http.Request) {
	if err := s.backups.DeleteSpace(sp(r).id, chi.URLParam(r, "name")); err != nil {
		writeBackupError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// spacesImport creates a new space, owned by the caller, from an uploaded
// space backup (or a Laravel database). Guests cannot.
func (s *Server) spacesImport(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	if u.IsGuest() {
		writeMessage(w, http.StatusForbidden, "Guests cannot create spaces.")
		return
	}
	tmp, name, ok := s.receiveUpload(w, r, s.spaceUploadLimit(r))
	if !ok {
		return
	}
	defer os.Remove(tmp)
	// Keep the extension: it tells an archive from a bare database file.
	src := tmp + filepath.Ext(name)
	if err := os.Rename(tmp, src); err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer os.Remove(src)
	space, err := s.backups.ImportSpace(r.Context(), s.spaces, src, r.FormValue("name"), u, s.spaceQuota(r))
	if err != nil {
		writeBackupError(w, err)
		return
	}
	writeData(w, http.StatusCreated, map[string]any{"id": space.ID, "uuid": space.UUID, "name": space.Name, "role": space.Role})
}

// requireSpaceAdmin lets only admins of the request's space through.
func (s *Server) requireSpaceAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if scope := sp(r); scope == nil || scope.role != domain.SpaceAdmin {
			writeMessage(w, http.StatusForbidden, "Only space administrators can do this.")
			return
		}
		next.ServeHTTP(w, r)
	})
}
