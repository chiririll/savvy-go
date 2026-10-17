// Go Savvy is a self-hosted expense tracker; see cmd for how the process runs.
package main

import (
	"savvy-go/cmd"
	"savvy-go/internal/appconfig"
)

// Set at link time: -ldflags "-X main.Version=… -X main.Env=…". They are handed down from
// here, so no package reads them as globals.
var (
	// Version is the release identifier exposed on health probes and the SPA shell.
	Version = "dev"
	// Env is the build environment (development, production, …).
	Env = "development"
)

func main() { cmd.Main(appconfig.Build{Version: Version, Env: Env}) }
