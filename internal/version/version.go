package version

// Value is the release identifier exposed on health probes and the SPA shell.
// Overridden at link time with -ldflags "-X savvy-go/internal/version.Value=…"
var Value = "dev"

// Env is the build environment (development, production, …).
// Overridden at link time with -ldflags "-X savvy-go/internal/version.Env=…"
var Env = "development"
