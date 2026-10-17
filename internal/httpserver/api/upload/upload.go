// Package upload serves the multipart upload routes.
package upload

import (
	"encoding/json"
	"io"
	"net/http"
	"savvy-go/internal/httpserver/httpx"
	"strconv"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/uploads"
)

func (s Handler) uploadCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Bucket   string  `json:"bucket"`
		Filename string  `json:"filename"`
		Type     *string `json:"type"`
		Size     *int64  `json:"size"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Bucket == "" || body.Filename == "" {
		httpx.WriteValidation(w, map[string][]string{"bucket": {"The bucket field is required."}})
		return
	}
	u := httpx.User(r)
	up, err := s.Uploads.Create(r.Context(), u.ID, body.Bucket, body.Filename, body.Size, body.Type)
	if err != nil {
		if err.Error() == "unknown bucket" {
			httpx.WriteValidation(w, map[string][]string{"bucket": {"The selected bucket is invalid."}})
			return
		}
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	var exp *string
	if up.ExpiresAt != nil {
		s := up.ExpiresAt.UTC().Format("2006-01-02T15:04:05-07:00")
		exp = &s
	}
	httpx.WriteJSON(w, http.StatusOK, createdBody{
		UploadID: up.ID, Key: up.ObjectKey, Bucket: up.Bucket, PartSize: up.PartSize, ExpiresAt: exp,
	})
}

// createdBody, completedBody and storedPartBody follow the multipart upload
// protocol the uploader client speaks, so their keys are not all camelCase.
type createdBody struct {
	UploadID  string  `json:"uploadId"`
	Key       string  `json:"key"`
	Bucket    string  `json:"bucket"`
	PartSize  int     `json:"partSize"`
	ExpiresAt *string `json:"expiresAt"`
}

type completedBody struct {
	Location string `json:"location"`
	Key      string `json:"key"`
	UploadID string `json:"uploadId"`
}

type storedPartBody struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
	Size       int64  `json:"size"`
}

func (s Handler) uploadSignPart(w http.ResponseWriter, r *http.Request) {
	up := s.uploadOwned(w, r)
	if up == nil {
		return
	}
	part, _ := strconv.Atoi(chi.URLParam(r, "part"))
	signed, err := s.Uploads.SignPart(up, part)
	if err != nil {
		httpx.WriteMessage(w, http.StatusConflict, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, signed)
}

func (s Handler) uploadListParts(w http.ResponseWriter, r *http.Request) {
	up := s.uploadOwned(w, r)
	if up == nil {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.Uploads.ListParts(up))
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
		httpx.WriteValidation(w, map[string][]string{"parts": {"The parts field is required."}})
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
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	httpx.WriteJSON(
		w,
		http.StatusOK,
		completedBody{Location: s.Uploads.Location(out), Key: out.ObjectKey, UploadID: out.ID},
	)
}

func (s Handler) uploadAbort(w http.ResponseWriter, r *http.Request) {
	up := s.uploadOwned(w, r)
	if up == nil {
		return
	}
	_ = s.Uploads.Abort(r.Context(), up)
	httpx.WriteJSON(w, http.StatusOK, struct{}{})
}

func (s Handler) uploadPart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	part, _ := strconv.Atoi(chi.URLParam(r, "part"))
	expires, _ := strconv.ParseInt(r.URL.Query().Get("expires"), 10, 64)
	sig := r.URL.Query().Get("signature")
	if !s.Uploads.VerifyPart(id, part, expires, sig) {
		httpx.WriteMessage(w, http.StatusForbidden, "Invalid signature.")
		return
	}
	up, _ := s.Uploads.ByID(r.Context(), id)
	if up == nil {
		httpx.WriteMessage(w, http.StatusNotFound, "Not found.")
		return
	}
	etag, size, err := s.Uploads.StorePart(up, part, io.LimitReader(r.Body, 64<<20))
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	w.Header().Set("ETag", `"`+etag+`"`)
	httpx.WriteJSON(w, http.StatusOK, storedPartBody{PartNumber: part, ETag: etag, Size: size})
}

func (s Handler) uploadOwned(w http.ResponseWriter, r *http.Request) *uploads.Upload {
	id := chi.URLParam(r, "upload")
	up, _ := s.Uploads.ByID(r.Context(), id)
	if up == nil {
		httpx.WriteMessage(w, http.StatusNotFound, "Not found.")
		return nil
	}
	u := httpx.User(r)
	if up.UserID == nil || *up.UserID != u.ID {
		httpx.WriteMessage(w, http.StatusForbidden, "Forbidden")
		return nil
	}
	if key := r.URL.Query().Get("key"); key != "" && key != up.ObjectKey {
		httpx.WriteMessage(w, http.StatusNotFound, "Not found.")
		return nil
	}
	return up
}
