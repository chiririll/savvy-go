package httpserver

import (
	"strings"
	"time"

	"savvy-go/internal/config"
)

// DefaultListen is the listen address of a new config file. A build may set it with
// -ldflags "-X savvy-go/internal/httpserver.DefaultListen=:80" (the container image does);
// it is loopback only otherwise.
var DefaultListen = "localhost:8080"

// serverTable is the [server] table. An empty value means "decide at run time".
type serverTable struct {
	Listen   string `toml:"listen" comment:"Address to listen on. Loopback only by default; use \":8080\" to listen on every interface."`
	AppURL   string `toml:"app_url" comment:"Public URL of the instance, e.g. \"https://savvy.example.com\". SSO and passkeys stay disabled without it."`
	Timezone string `toml:"timezone" comment:"Time zone."`
}

func (serverTable) Defaults() serverTable {
	return serverTable{Listen: DefaultListen, Timezone: "UTC"}
}

// ServerConfig is the [server] section.
type ServerConfig struct {
	Listen string
	// AppURL is the public URL without a trailing slash, empty when unset.
	AppURL   string
	TZ       string
	Location *time.Location
}

// decodeServer fills what the file leaves empty. An unknown time zone is UTC.
func decodeServer(t serverTable) (ServerConfig, error) {
	tz := firstNonEmpty(t.Timezone, "UTC")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc, tz = time.UTC, "UTC"
	}
	return ServerConfig{
		Listen:   firstNonEmpty(t.Listen, DefaultListen),
		AppURL:   strings.TrimRight(strings.TrimSpace(t.AppURL), "/"),
		TZ:       tz,
		Location: loc,
	}, nil
}

// ServerSection is the [server] table of the config file.
var ServerSection = config.Section[serverTable, ServerConfig]{Name: "server", Decode: decodeServer}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
