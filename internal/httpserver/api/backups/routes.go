package backups

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	routing.Register(spec, func(d *routing.Deps) []routing.Mount {
		h := Handler{Cfg: d.Cfg, Store: d.Store, Backups: d.Backups, Spaces: d.Spaces, Settings: d.Settings, Users: d.Users, Bootstrap: d.Bootstrap, Guard: d.Guard}
		return []routing.Mount{
			routing.Public(h.PublicRoutes),
			routing.Account(h.AccountRoutes),
			routing.SpaceAdmin(h.SpaceRoutes),
		}
	})
}

func (h Handler) SpaceRoutes(r chi.Router) {
	// P1: a space's backups are for its admins, never for API tokens.
	r.Get("/backups", h.spaceBackupsIndex)
	r.Post("/backups", h.spaceBackupsStore)
	r.Post("/backups/upload", h.spaceBackupsUpload)
	r.Post("/backups/upgrade-laravel", h.spaceBackupsUpgradeLaravel)
	r.Get("/backups/{name}/download", h.spaceBackupsDownload)
	r.Post("/backups/{name}/restore", h.spaceBackupsRestore)
	r.Delete("/backups/{name}", h.spaceBackupsDestroy)
}

func (h Handler) AccountRoutes(r chi.Router) {
	r.With(h.Guard.SessionOnly).Post("/spaces/import", h.spacesImport)
}

func (h Handler) PublicRoutes(r chi.Router) {
	r.Post("/onboarding/restore", h.onboardingRestore)
}
