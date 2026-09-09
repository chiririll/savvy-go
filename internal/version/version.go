package version

// Value is the release identifier exposed on health probes and the SPA shell.
// Overridden at link time with -ldflags "-X github.com/chiririll/savvy-plus/internal/version.Value=…"
var Value = "dev"
