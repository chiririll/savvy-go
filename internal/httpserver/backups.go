package httpserver

import "net/http"

// Backups are being rebuilt for the per-space layout (a server backup holds
// server.sqlite and every space database; each space gets its own). Until
// then the endpoints refuse instead of copying a single database file.
func (s *Server) backupsUnavailable(w http.ResponseWriter, _ *http.Request) {
	writeMessage(w, http.StatusNotImplemented, "Backups are temporarily unavailable.")
}

func (s *Server) backupsIndex(w http.ResponseWriter, r *http.Request)    { s.backupsUnavailable(w, r) }
func (s *Server) backupsStore(w http.ResponseWriter, r *http.Request)    { s.backupsUnavailable(w, r) }
func (s *Server) backupsUpload(w http.ResponseWriter, r *http.Request)   { s.backupsUnavailable(w, r) }
func (s *Server) backupsDownload(w http.ResponseWriter, r *http.Request) { s.backupsUnavailable(w, r) }
func (s *Server) backupsRestore(w http.ResponseWriter, r *http.Request)  { s.backupsUnavailable(w, r) }
func (s *Server) backupsDestroy(w http.ResponseWriter, r *http.Request)  { s.backupsUnavailable(w, r) }
