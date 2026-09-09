package httpserver

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) importParse(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UploadID string `json:"upload_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UploadID == "" {
		writeValidation(w, map[string][]string{"upload_id": {"The upload id field is required."}})
		return
	}
	up, _ := s.uploads.ByID(r.Context(), body.UploadID)
	u := userFrom(r)
	if up == nil || up.UserID == nil || *up.UserID != u.ID {
		writeMessage(w, http.StatusForbidden, "Forbidden")
		return
	}
	if up.Bucket != "transaction-imports" || up.Status != "completed" {
		writeMessage(w, 422, "The uploaded file is not ready.")
		return
	}
	im, err := s.imports.Create(r.Context(), u.ID, up.ID)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	payload := im.JSON()
	s.runJob(func(ctx context.Context) {
		_ = s.imports.Parse(ctx, im.ID, up.ID)
	})
	writeData(w, http.StatusOK, payload)
}

func (s *Server) importShow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "import")
	im, _ := s.imports.ByID(r.Context(), id)
	u := userFrom(r)
	if im == nil {
		writeMessage(w, http.StatusNotFound, "Not found.")
		return
	}
	if im.UserID == nil || *im.UserID != u.ID {
		writeMessage(w, http.StatusForbidden, "Forbidden")
		return
	}
	writeData(w, http.StatusOK, im.JSON())
}

func (s *Server) importPreview(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ImportID string         `json:"import_id"`
		Mapping  map[string]any `json:"mapping"`
		Options  map[string]any `json:"options"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ImportID == "" {
		writeValidation(w, map[string][]string{"import_id": {"The import id field is required."}})
		return
	}
	im, _ := s.imports.ByID(r.Context(), body.ImportID)
	u := userFrom(r)
	if im == nil || im.UserID == nil || *im.UserID != u.ID {
		writeMessage(w, http.StatusForbidden, "Forbidden")
		return
	}
	result, err := s.imports.Preview(r.Context(), im, body.Mapping, body.Options)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusOK, result)
}

func (s *Server) importExecute(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ImportID string         `json:"import_id"`
		Mapping  map[string]any `json:"mapping"`
		Options  map[string]any `json:"options"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ImportID == "" {
		writeValidation(w, map[string][]string{"import_id": {"The import id field is required."}})
		return
	}
	im, _ := s.imports.ByID(r.Context(), body.ImportID)
	u := userFrom(r)
	if im == nil || im.UserID == nil || *im.UserID != u.ID {
		writeMessage(w, http.StatusForbidden, "Forbidden")
		return
	}
	im.Status = "importing"
	im.Mapping = body.Mapping
	im.Options = body.Options
	payload := im.JSON()
	s.runJob(func(ctx context.Context) {
		_ = s.imports.Execute(ctx, im.ID, body.Mapping, body.Options)
	})
	writeData(w, http.StatusOK, payload)
}

func (s *Server) runJob(fn func(context.Context)) {
	if s.queue != nil {
		s.queue.Enqueue(fn)
		return
	}
	fn(context.Background())
}
