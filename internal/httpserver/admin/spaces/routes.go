package spaces

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx/routing"
)

func init() {
	routing.Register(nil, func(d *routing.Deps) []routing.Mount {
		h := Handler{Spaces: d.Spaces, Users: d.Users, Backups: d.Backups}
		return []routing.Mount{
			routing.AdminSession(h.Routes),
		}
	})
}

func (h Handler) Routes(r chi.Router) {
	r.Post("/admin/invitations/clear", h.invitationsClear)
	r.Get("/admin/spaces", h.adminSpacesIndex)
	r.Patch("/admin/spaces/{id}", h.adminSpaceUpdate)
	r.Delete("/admin/spaces/{id}", h.adminSpaceDestroy)
	r.Post("/admin/spaces/{id}/admins", h.adminSpaceAssignAdmin)
	r.Get("/admin/deleted-spaces", h.adminDeletedIndex)
	r.Post("/admin/deleted-spaces/{name}/restore", h.adminDeletedRestore)
}
