package httpserver

// The handler packages only server admins use.
import (
	_ "savvy-go/internal/httpserver/admin/backups"
	_ "savvy-go/internal/httpserver/admin/checks"
	_ "savvy-go/internal/httpserver/admin/keys"
	_ "savvy-go/internal/httpserver/admin/mail"
	_ "savvy-go/internal/httpserver/admin/monitoring"
	_ "savvy-go/internal/httpserver/admin/oauth"
	_ "savvy-go/internal/httpserver/admin/spaces"
	_ "savvy-go/internal/httpserver/admin/sso"
	_ "savvy-go/internal/httpserver/admin/users"
)
