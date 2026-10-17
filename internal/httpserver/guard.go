package httpserver

import (
	"savvy-go/internal/app/signin"
	"savvy-go/internal/appconfig"
	"savvy-go/internal/auth"
	"savvy-go/internal/httpserver/httpx/guard"
	"savvy-go/internal/locator"
	"savvy-go/internal/refsync"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
)

func (s *Server) newGuard() *guard.Guard {
	return &guard.Guard{
		Cfg:           locator.Get[appconfig.Config](s.loc),
		Security:      locator.Get[auth.Security](s.loc),
		Sessions:      locator.Get[auth.Sessions](s.loc),
		APITokens:     locator.Get[auth.APITokens](s.loc),
		EmailRequired: locator.Get[signin.Policy](s.loc).EmailRequired,
		Space: guard.SpaceDeps{
			Spaces:   locator.Get[spaces.Service](s.loc),
			Store:    locator.Get[store.Store](s.loc),
			Refsync:  locator.Get[refsync.Service](s.loc),
			NewScope: s.newScope,
		},
	}
}
