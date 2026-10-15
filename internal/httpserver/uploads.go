package httpserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/uploads"
)

func (s *Server) uploadCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Bucket   string  `json:"bucket"`
		Filename string  `json:"filename"`
		Type     *string `json:"type"`
		Size     *int64  `json:"size"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Bucket == "" || body.Filename == "" {
		writeValidation(w, map[string][]string{"bucket": {"The bucket field is required."}})
		return
	}
	u := userFrom(r)
	up, err := s.uploads.Create(r.Context(), u.ID, body.Bucket, body.Filename, body.Size, body.Type)
	if err != nil {
		if err.Error() == "unknown bucket" {
			writeValidation(w, map[string][]string{"bucket": {"The selected bucket is invalid."}})
			return
		}
		writeMessage(w, 422, err.Error())
		return
	}
	var exp any
	if up.ExpiresAt != nil {
		exp = up.ExpiresAt.UTC().Format("2006-01-02T15:04:05-07:00")
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"uploadId": up.ID, "key": up.ObjectKey, "bucket": up.Bucket,
		"partSize": up.PartSize, "expiresAt": exp,
	})
}

func (s *Server) uploadSignPart(w http.ResponseWriter, r *http.Request) {
	up := s.uploadOwned(w, r)
	if up == nil {
		return
	}
	part, _ := strconv.Atoi(chi.URLParam(r, "part"))
	signed, err := s.uploads.SignPart(up, part)
	if err != nil {
		writeMessage(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, signed)
}

func (s *Server) uploadListParts(w http.ResponseWriter, r *http.Request) {
	up := s.uploadOwned(w, r)
	if up == nil {
		return
	}
	writeJSON(w, http.StatusOK, s.uploads.ListParts(up))
}

func (s *Server) uploadComplete(w http.ResponseWriter, r *http.Request) {
	up := s.uploadOwned(w, r)
	if up == nil {
		return
	}
	var body struct {
		Parts []struct {
			PartNumber int    `json:"PartNumber"`
			ETag       string `json:"ETag"`
		} `json:"parts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Parts) == 0 {
		writeValidation(w, map[string][]string{"parts": {"The parts field is required."}})
		return
	}
	var parts []struct {
		Number int
		ETag   string
	}
	for _, p := range body.Parts {
		parts = append(parts, struct {
			Number int
			ETag   string
		}{p.PartNumber, p.ETag})
	}
	out, err := s.uploads.Complete(r.Context(), up, parts)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"location": s.uploads.Location(out), "key": out.ObjectKey, "uploadId": out.ID,
	})
}

func (s *Server) uploadAbort(w http.ResponseWriter, r *http.Request) {
	up := s.uploadOwned(w, r)
	if up == nil {
		return
	}
	_ = s.uploads.Abort(r.Context(), up)
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) uploadPart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	part, _ := strconv.Atoi(chi.URLParam(r, "part"))
	expires, _ := strconv.ParseInt(r.URL.Query().Get("expires"), 10, 64)
	sig := r.URL.Query().Get("signature")
	if !s.uploads.VerifyPart(id, part, expires, sig) {
		writeMessage(w, http.StatusForbidden, "Invalid signature.")
		return
	}
	up, _ := s.uploads.ByID(r.Context(), id)
	if up == nil {
		writeMessage(w, http.StatusNotFound, "Not found.")
		return
	}
	etag, size, err := s.uploads.StorePart(up, part, io.LimitReader(r.Body, 64<<20))
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	w.Header().Set("ETag", `"`+etag+`"`)
	writeJSON(w, http.StatusOK, map[string]any{"part_number": part, "etag": etag, "size": size})
}

func (s *Server) uploadOwned(w http.ResponseWriter, r *http.Request) *uploads.Upload {
	id := chi.URLParam(r, "upload")
	up, _ := s.uploads.ByID(r.Context(), id)
	if up == nil {
		writeMessage(w, http.StatusNotFound, "Not found.")
		return nil
	}
	u := userFrom(r)
	if up.UserID == nil || *up.UserID != u.ID {
		writeMessage(w, http.StatusForbidden, "Forbidden")
		return nil
	}
	if key := r.URL.Query().Get("key"); key != "" && key != up.ObjectKey {
		writeMessage(w, http.StatusNotFound, "Not found.")
		return nil
	}
	return up
}

func (s *Server) publicUploadsRoutes(r chi.Router) {
	r.Put("/uploads/{id}/parts/{part}", s.uploadPart)
}

func (s *Server) uploadRoutes(r chi.Router) {
	r.Post("/s3/multipart", s.uploadCreate)
	r.Get("/s3/multipart/{upload}", s.uploadListParts)
	r.Get("/s3/multipart/{upload}/{part}", s.uploadSignPart)
	r.Post("/s3/multipart/{upload}/complete", s.uploadComplete)
	r.Delete("/s3/multipart/{upload}", s.uploadAbort)
}

// randomSecret is a fresh secret for signing upload part URLs. It lives as long
// as the process: those URLs expire within minutes, and an upload interrupted
// by a restart starts over.
func randomSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err) // the system's random source is broken: nothing safe to do
	}
	return hex.EncodeToString(b)
}
