package auditlog

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/httpserver/ui"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	ui.AddSpec(spec)
	routing.Register(func(*routing.Deps) []routing.Mount {
		h := Handler{}
		return []routing.Mount{
			routing.SpaceRead(h.ReadRoutes),
			routing.SpaceWrite(h.WriteRoutes),
			routing.SpaceAdmin(h.AdminRoutes),
		}
	})
}

// Handler serves /audit-log of a space.
type Handler struct{}

// ReadRoutes: any member reads; what they see depends on who they are (see list).
func (h Handler) ReadRoutes(r chi.Router) {
	r.Get("/audit-log", h.list)
	r.Get("/audit-log/{id}", h.show)
}

// WriteRoutes: rolling back. A token rolls back its own entries; in the
// browser only a space admin rolls back, and any entry.
func (h Handler) WriteRoutes(r chi.Router) {
	r.Post("/audit-log/rollback", h.rollback)
}

// AdminRoutes: clearing the log is for a space admin in the browser, never a token.
func (h Handler) AdminRoutes(r chi.Router) {
	r.Delete("/audit-log", h.clear)
}
