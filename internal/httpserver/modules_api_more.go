package httpserver

// One handler package per resource of the space API (imports to uploads).
import (
	_ "savvy-go/internal/httpserver/api/imports"
	_ "savvy-go/internal/httpserver/api/recurring"
	_ "savvy-go/internal/httpserver/api/reports"
	_ "savvy-go/internal/httpserver/api/settings"
	_ "savvy-go/internal/httpserver/api/space"
	_ "savvy-go/internal/httpserver/api/syncgroups"
	_ "savvy-go/internal/httpserver/api/tags"
	_ "savvy-go/internal/httpserver/api/transactions"
	_ "savvy-go/internal/httpserver/api/transfer"
	_ "savvy-go/internal/httpserver/api/upload"
)
