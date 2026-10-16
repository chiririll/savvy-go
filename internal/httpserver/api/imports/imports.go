// Package imports serves the transaction import of a space.
package imports

import (
	"context"
	"encoding/json"
	"net/http"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/jobs"
	"savvy-go/internal/uploads"

	"savvy-go/internal/httpserver/httpx/dto"

	"github.com/go-chi/chi/v5"
)

// Handler serves the import routes. Jobs run on Queue, or at once when it is nil.
type Handler struct {
	Uploads uploads.Service
	Queue   *jobs.Queue
}

func (s Handler) importParse(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UploadID string `json:"upload_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UploadID == "" {
		httpx.WriteValidation(w, map[string][]string{"upload_id": {"The upload id field is required."}})
		return
	}
	up, _ := s.Uploads.ByID(r.Context(), body.UploadID)
	u := httpx.User(r)
	if up == nil || up.UserID == nil || *up.UserID != u.ID {
		httpx.WriteMessage(w, http.StatusForbidden, "Forbidden")
		return
	}
	if up.Bucket != "transaction-imports" || up.Status != "completed" {
		httpx.WriteMessage(w, 422, "The uploaded file is not ready.")
		return
	}
	im, err := httpx.Scope(r).Imports.Create(r.Context(), u.ID, up.ID)
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	payload := dto.NewImport(*im)
	s.runJob(func(ctx context.Context) {
		_ = httpx.Scope(r).Imports.Parse(ctx, im.ID, up.ID)
	})
	httpx.WriteData(w, http.StatusOK, payload)
}

func (s Handler) importShow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "import")
	im, _ := httpx.Scope(r).Imports.ByID(r.Context(), id)
	u := httpx.User(r)
	if im == nil {
		httpx.WriteMessage(w, http.StatusNotFound, "Not found.")
		return
	}
	if im.UserID == nil || *im.UserID != u.ID {
		httpx.WriteMessage(w, http.StatusForbidden, "Forbidden")
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewImport(*im))
}

func (s Handler) importPreview(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ImportID string         `json:"import_id"`
		Mapping  map[string]any `json:"mapping"`
		Options  map[string]any `json:"options"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ImportID == "" {
		httpx.WriteValidation(w, map[string][]string{"import_id": {"The import id field is required."}})
		return
	}
	im, _ := httpx.Scope(r).Imports.ByID(r.Context(), body.ImportID)
	u := httpx.User(r)
	if im == nil || im.UserID == nil || *im.UserID != u.ID {
		httpx.WriteMessage(w, http.StatusForbidden, "Forbidden")
		return
	}
	result, err := httpx.Scope(r).Imports.Preview(r.Context(), im, body.Mapping, body.Options)
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, result)
}

func (s Handler) importExecute(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ImportID string         `json:"import_id"`
		Mapping  map[string]any `json:"mapping"`
		Options  map[string]any `json:"options"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ImportID == "" {
		httpx.WriteValidation(w, map[string][]string{"import_id": {"The import id field is required."}})
		return
	}
	im, _ := httpx.Scope(r).Imports.ByID(r.Context(), body.ImportID)
	u := httpx.User(r)
	if im == nil || im.UserID == nil || *im.UserID != u.ID {
		httpx.WriteMessage(w, http.StatusForbidden, "Forbidden")
		return
	}
	im.Status = "importing"
	im.Mapping = body.Mapping
	im.Options = body.Options
	payload := dto.NewImport(*im)
	s.runJob(func(ctx context.Context) {
		_ = httpx.Scope(r).Imports.Execute(ctx, im.ID, body.Mapping, body.Options)
	})
	httpx.WriteData(w, http.StatusOK, payload)
}

func (s Handler) runJob(fn func(context.Context)) {
	if s.Queue != nil {
		s.Queue.Enqueue(fn)
		return
	}
	fn(context.Background())
}
