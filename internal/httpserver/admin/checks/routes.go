package checks

import (
	"github.com/go-chi/chi/v5"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/backup"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/mail"
)

func init() {
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{
			Cfg:        routing.Get[appconfig.Config](d),
			Writable:   d.ConfigWritable,
			UploadsDir: routing.Get[appconfig.Config](d).UploadsDir,
			Backups:    routing.Get[backup.Service](d),
			Mail:       routing.Get[*mail.Service](d).Status,
		}
		return []routing.Mount{
			routing.Admin(h.Routes),
		}
	})
}

// Handler lists the checks of a running server.
type Handler struct {
	Cfg        appconfig.Config
	Writable   func() bool // whether the config file can be changed
	UploadsDir string
	Backups    backup.Service
	// Mail reports whether mail works, and what is wrong with its settings.
	Mail func() (enabled bool, problem string)
}

func (h Handler) Routes(r chi.Router) {
	r.Get("/admin/checks", h.index)
}
