package upload

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/uploads"
)

func init() {
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{Uploads: routing.Get[uploads.Service](d)}
		return []routing.Mount{
			routing.Public(h.PublicRoutes),
			routing.SpaceWrite(h.Routes),
		}
	})
}

// Handler serves the upload routes.
type Handler struct{ Uploads uploads.Service }

func (s Handler) PublicRoutes(r chi.Router) {
	r.Put("/uploads/{id}/parts/{part}", s.uploadPart)
}

func (s Handler) Routes(r chi.Router) {
	r.Post("/s3/multipart", s.uploadCreate)
	r.Get("/s3/multipart/{upload}", s.uploadListParts)
	r.Get("/s3/multipart/{upload}/{part}", s.uploadSignPart)
	r.Post("/s3/multipart/{upload}/complete", s.uploadComplete)
	r.Delete("/s3/multipart/{upload}", s.uploadAbort)
}
