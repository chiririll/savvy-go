// Package backups serves server and space backups, restoring from them, and the first-run restore.
package backups

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/auth"
	"savvy-go/internal/backup"
	"savvy-go/internal/bootstrap"
	"savvy-go/internal/config"
	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/guard"
	"savvy-go/internal/settings"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
)

// Server backups hold every database and the signing key: server admins only
// (requireAdmin on the routes). Space backups hold one space: its admins.

const maxServerUpload = 4 << 30 // 4 GiB

// Handler holds what the backup routes need.
type Handler struct {
	Cfg       config.Config
	Store     store.Store
	Backups   backup.Service
	Spaces    spaces.Service
	Settings  settings.Store
	Users     auth.Users
	Bootstrap *bootstrap.Guard
	Guard     *guard.Guard
}

func (h Handler) backupsService(r *http.Request) backup.Service {
	b := h.Backups
	b.KeepPerSpace = int(h.Settings.Int(r.Context(), "space_backups_max", 10))
	return b
}

// spaceQuota is the size limit of the request's space, or the default limit
// for a new space outside one.
func (h Handler) spaceQuota(r *http.Request) int64 {
	if scope := api.Scope(r); scope != nil {
		return h.Spaces.Quota(r.Context(), scope.ID)
	}
	return settings.SpaceQuota(r.Context(), h.Store.Server())
}

func (h Handler) AdminRoutes(r chi.Router) {
	// P1: backups hold every space.
	r.Get("/backups", h.backupsIndex)
	r.Post("/backups", h.backupsStore)
	r.Post("/backups/upload", h.backupsUpload)
	r.Post("/backups/upgrade-laravel", h.backupsUpgradeLaravel)
	r.Get("/backups/{name}/download", h.backupsDownload)
	r.Post("/backups/{name}/restore", h.backupsRestore)
	r.Delete("/backups/{name}", h.backupsDestroy)
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
