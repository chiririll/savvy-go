// Package backups serves the backups of a space, importing a space, and the first-run restore.
package backups

import (
	"net/http"

	"savvy-go/internal/auth"
	"savvy-go/internal/backup"
	"savvy-go/internal/bootstrap"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/guard"
	"savvy-go/internal/settings"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
)

// Server backups hold every database and the signing key: server admins only
// (requireAdmin on the routes). Space backups hold one space: its admins.

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

func (h Handler) backupsService(r *http.Request) backup.Service {
	b := h.Backups
	b.KeepPerSpace = int(h.Settings.Int(r.Context(), "space_backups_max", 10))
	return b
}

// spaceQuota is the size limit of the request's space, or the default limit
// for a new space outside one.
func (h Handler) spaceQuota(r *http.Request) int64 {
	if scope := httpx.Scope(r); scope != nil {
		return h.Spaces.Quota(r.Context(), scope.ID)
	}
	return settings.SpaceQuota(r.Context(), h.Store.Server())
}
