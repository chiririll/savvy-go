// Package backups serves server and space backups, restoring from them, and the first-run restore.
package backups

import (
	"net/http"

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
