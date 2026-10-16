package syncgroups

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/httpserver/ui"
	"savvy-go/internal/refsync"
	"savvy-go/internal/spaces"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	ui.AddSpec(spec)
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{Spaces: routing.Get[spaces.Service](d), Refsync: routing.Get[refsync.Service](d), Guard: d.Guard}
		return []routing.Mount{
			routing.Account(h.AccountRoutes),
			routing.SpaceRead(h.ReadRoutes),
		}
	})
}

func (h Handler) ReadRoutes(r chi.Router) {
	r.Get("/sync-group", h.syncGroupShow)
}

func (h Handler) AccountRoutes(r chi.Router) {
	// Sync groups span several spaces; the rights are checked in each.
	r.With(h.Guard.SessionOnly).Post("/sync-groups/diff", h.syncGroupDiff)
	r.With(h.Guard.SessionOnly).Post("/sync-groups/apply", h.syncGroupApply)
}
