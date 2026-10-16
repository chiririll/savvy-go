package uploadapi

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/routing"
)

func init() {
	routing.Register(nil, func(d *routing.Deps) []routing.Mount {
		h := Handler{Uploads: d.Uploads}
		return []routing.Mount{
			routing.Public(h.PublicRoutes),
			routing.SpaceWrite(h.Routes),
		}
	})
}

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
