package backups

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/backup"
	"savvy-go/internal/httpserver/httpx/routing"
)

func init() {
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{BackupsDir: routing.Get[appconfig.Config](d).BackupsDir, Backups: routing.Get[backup.Service](d)}
		return []routing.Mount{
			routing.AdminSession(h.Routes),
		}
	})
}

func (h Handler) Routes(r chi.Router) {
	// P1: backups hold every space.
	r.Get("/backups", h.backupsIndex)
	r.Post("/backups", h.backupsStore)
	r.Post("/backups/upload", h.backupsUpload)
	r.Post("/backups/upgrade-laravel", h.backupsUpgradeLaravel)
	r.Get("/backups/{name}/download", h.backupsDownload)
	r.Post("/backups/{name}/restore", h.backupsRestore)
	r.Delete("/backups/{name}", h.backupsDestroy)
}
