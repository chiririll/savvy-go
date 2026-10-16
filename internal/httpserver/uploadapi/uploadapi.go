// Package uploadapi serves the multipart upload routes.
package uploadapi

import (
	"encoding/json"
	"io"
	"net/http"
	"savvy-go/internal/httpserver/api"
	"strconv"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/uploads"
)

// Handler serves the upload routes.
type Handler struct{ Uploads uploads.Service }

func (s Handler) uploadCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Bucket   string  `json:"bucket"`
		Filename string  `json:"filename"`
		Type     *string `json:"type"`
		Size     *int64  `json:"size"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Bucket == "" || body.Filename == "" {
		api.WriteValidation(w, map[string][]string{"bucket": {"The bucket field is required."}})
		return
	}
	u := api.User(r)
	up, err := s.Uploads.Create(r.Context(), u.ID, body.Bucket, body.Filename, body.Size, body.Type)
	if err != nil {
		if err.Error() == "unknown bucket" {
			api.WriteValidation(w, map[string][]string{"bucket": {"The selected bucket is invalid."}})
			return
		}
		api.WriteMessage(w, 422, err.Error())
		return
	}
	var exp any
	if up.ExpiresAt != nil {
		exp = up.ExpiresAt.UTC().Format("2006-01-02T15:04:05-07:00")
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"uploadId": up.ID, "key": up.ObjectKey, "bucket": up.Bucket,
		"partSize": up.PartSize, "expiresAt": exp,
	})
}

func (s Handler) uploadSignPart(w http.ResponseWriter, r *http.Request) {
	up := s.uploadOwned(w, r)
	if up == nil {
		return
	}
	part, _ := strconv.Atoi(chi.URLParam(r, "part"))
	signed, err := s.Uploads.SignPart(up, part)
	if err != nil {
		api.WriteMessage(w, http.StatusConflict, err.Error())
		return
	}
	api.WriteJSON(w, http.StatusOK, signed)
}

func (s Handler) uploadListParts(w http.ResponseWriter, r *http.Request) {
	up := s.uploadOwned(w, r)
	if up == nil {
		return
	}
	api.WriteJSON(w, http.StatusOK, s.Uploads.ListParts(up))
}

func (s Handler) uploadComplete(w http.ResponseWriter, r *http.Request) {
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
		api.WriteValidation(w, map[string][]string{"parts": {"The parts field is required."}})
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
	out, err := s.Uploads.Complete(r.Context(), up, parts)
	if err != nil {
		api.WriteMessage(w, 422, err.Error())
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"location": s.Uploads.Location(out), "key": out.ObjectKey, "uploadId": out.ID,
	})
}

func (s Handler) uploadAbort(w http.ResponseWriter, r *http.Request) {
	up := s.uploadOwned(w, r)
	if up == nil {
		return
	}
	_ = s.Uploads.Abort(r.Context(), up)
	api.WriteJSON(w, http.StatusOK, map[string]any{})
}

func (s Handler) uploadPart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	part, _ := strconv.Atoi(chi.URLParam(r, "part"))
	expires, _ := strconv.ParseInt(r.URL.Query().Get("expires"), 10, 64)
	sig := r.URL.Query().Get("signature")
	if !s.Uploads.VerifyPart(id, part, expires, sig) {
		api.WriteMessage(w, http.StatusForbidden, "Invalid signature.")
		return
	}
	up, _ := s.Uploads.ByID(r.Context(), id)
	if up == nil {
		api.WriteMessage(w, http.StatusNotFound, "Not found.")
		return
	}
	etag, size, err := s.Uploads.StorePart(up, part, io.LimitReader(r.Body, 64<<20))
	if err != nil {
		api.WriteMessage(w, 422, err.Error())
		return
	}
	w.Header().Set("ETag", `"`+etag+`"`)
	api.WriteJSON(w, http.StatusOK, map[string]any{"part_number": part, "etag": etag, "size": size})
}

func (s Handler) uploadOwned(w http.ResponseWriter, r *http.Request) *uploads.Upload {
	id := chi.URLParam(r, "upload")
	up, _ := s.Uploads.ByID(r.Context(), id)
	if up == nil {
		api.WriteMessage(w, http.StatusNotFound, "Not found.")
		return nil
	}
	u := api.User(r)
	if up.UserID == nil || *up.UserID != u.ID {
		api.WriteMessage(w, http.StatusForbidden, "Forbidden")
		return nil
	}
	if key := r.URL.Query().Get("key"); key != "" && key != up.ObjectKey {
		api.WriteMessage(w, http.StatusNotFound, "Not found.")
		return nil
	}
	return up
}
