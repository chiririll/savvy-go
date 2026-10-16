// Package backups serves the backups of a space, importing a space, and the first-run restore.
package backups

import (
	"net/http"

	"savvy-go/internal/backup"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/settings"
)

// Server backups hold every database and the signing key: server admins only
// (requireAdmin on the routes). Space backups hold one space: its admins.

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
