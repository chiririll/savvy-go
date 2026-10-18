package httpserver

// Every package that serves routes registers them from its routes.go; linking it in is
// what mounts them. Add a new handler package to the file for its kind.
import (
	_ "savvy-go/internal/httpserver/health"
	_ "savvy-go/internal/httpserver/ui"
)
