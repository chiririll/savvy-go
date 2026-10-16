package backups

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/auth"
	"savvy-go/internal/backup"
	"savvy-go/internal/bootstrap"
	"savvy-go/internal/httpserver/httpx/guard"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/httpserver/ui"
	"savvy-go/internal/settings"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	ui.AddSpec(spec)
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{BackupsDir: routing.Get[appconfig.Config](d).BackupsDir, Store: routing.Get[store.Store](d), Backups: routing.Get[backup.Service](d), Spaces: routing.Get[spaces.Service](d), Settings: routing.Get[settings.Store](d), Users: routing.Get[auth.Users](d), Bootstrap: routing.Get[*bootstrap.Guard](d), Guard: d.Guard}
		return []routing.Mount{
			routing.Public(h.PublicRoutes),
			routing.Account(h.AccountRoutes),
			routing.SpaceAdmin(h.SpaceRoutes),
		}
	})
}

// Handler holds what the backup routes need.
type Handler struct {
	BackupsDir string
	Store      store.Store
	Backups    backup.Service
	Spaces     spaces.Service
	Settings   settings.Store
	Users      auth.Users
	Bootstrap  *bootstrap.Guard
	Guard      *guard.Guard
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
