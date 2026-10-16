package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"savvy-go/internal/httpserver/accounts"
	"savvy-go/internal/httpserver/apidocs"
	"savvy-go/internal/httpserver/apitokens"
	"savvy-go/internal/httpserver/authapi"
	"savvy-go/internal/httpserver/automation"
	"savvy-go/internal/httpserver/backups"
	"savvy-go/internal/httpserver/budgets"
	"savvy-go/internal/httpserver/catalog"
	"savvy-go/internal/httpserver/categories"
	"savvy-go/internal/httpserver/checks"
	"savvy-go/internal/httpserver/currencies"
	"savvy-go/internal/httpserver/debts"
	"savvy-go/internal/httpserver/health"
	"savvy-go/internal/httpserver/imports"
	"savvy-go/internal/httpserver/mailing"
	"savvy-go/internal/httpserver/mcpserver"
	"savvy-go/internal/httpserver/monitoring"
	"savvy-go/internal/httpserver/oauthapi"
	"savvy-go/internal/httpserver/recurring"
	"savvy-go/internal/httpserver/reports"
	"savvy-go/internal/httpserver/settingsapi"
	"savvy-go/internal/httpserver/spaceapi"
	"savvy-go/internal/httpserver/spacefill"
	"savvy-go/internal/httpserver/ssoapi"
	"savvy-go/internal/httpserver/syncgroups"
	"savvy-go/internal/httpserver/tags"
	"savvy-go/internal/httpserver/transactions"
	"savvy-go/internal/httpserver/transferapi"
	"savvy-go/internal/httpserver/twofactor"
	"savvy-go/internal/httpserver/uploadapi"
	"savvy-go/internal/httpserver/users"
	"savvy-go/internal/httpserver/webauthn"
)

func (s *Server) routes() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(locale)
	r.Use(noStoreAPI)
	r.Use(noFraming)

	probes := health.Handler{Store: s.store}
	r.Get("/livez", probes.Live)
	r.Get("/readyz", probes.Ready)

	authHandler := authapi.Handler{
		Cfg: s.cfg, Users: s.users, Sessions: s.sessions, Tokens: s.tokens, Challenges: s.challenges,
		Settings: s.settings, Bootstrap: s.bootstrap, Login: s.login,
	}

	twoFactor := twofactor.Handler{TwoFactor: s.twoFactor, Users: s.users, Challenges: s.challenges, Login: s.login}
	passkeys := webauthn.Handler{WebAuthn: s.webauthn, Challenges: s.challenges, Login: s.login, Guard: s.guard}

	sso := ssoapi.Handler{Cfg: s.cfg, SSO: &s.sso, Users: s.users, Challenges: s.challenges, Login: s.login, Guard: s.guard}

	backupsH := s.backupsHandler()

	oauthH := oauthapi.Handler{
		Cfg: s.cfg, OAuth: s.oauth, Spaces: s.spaces,
		MCP: mcpserver.Handler{
			Cfg: s.cfg, APITokens: s.apiTokens,
			// The tools call the finished router, which exists after New.
			Inner: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }),
		}.Endpoint(),
	}

	spaces := s.spaceHandler()
	transferH := s.transferHandler()
	settingsH := s.settingsHandler()

	r.Route("/api", func(r chi.Router) {
		apidocs.Routes(r)

		authHandler.PublicRoutes(r)
		backupsH.PublicRoutes(r)
		s.verify.PublicRoutes(r)
		twoFactor.PublicRoutes(r)
		sso.PublicRoutes(r)
		passkeys.PublicRoutes(r)
		uploadapi.Handler{Uploads: s.uploads}.PublicRoutes(r)
		spaces.PublicRoutes(r)

		r.Group(func(r chi.Router) {
			r.Use(s.guard.RequireSession)
			r.Use(s.guard.RequireCSRF)
			r.Use(s.guard.RequireVerifiedEmail)

			// Credential management is limited to real sessions, never API tokens.
			r.Group(func(r chi.Router) {
				r.Use(s.guard.SessionOnly)
				authHandler.CredentialRoutes(r)
				s.verify.CredentialRoutes(r)
				twoFactor.CredentialRoutes(r)
				passkeys.CredentialRoutes(r)
				apitokens.Handler{Tokens: s.apiTokens, Spaces: s.spaces}.Routes(r)
				oauthH.CredentialRoutes(r)
			})

			catalog.Handler{Rates: s.rates}.Routes(r)
			spaces.AccountRoutes(r)
			backupsH.AccountRoutes(r)
			s.syncGroupsHandler().AccountRoutes(r)
			r.Route("/spaces/{space}", s.spaceRoutes)

			// P2: users, instance settings, monitoring and the space overview
			// are the server admins'.
			r.Group(func(r chi.Router) {
				r.Use(s.guard.RequireAdmin)
				s.usersHandler().Routes(r)
				sso.AdminRoutes(r)
				monitoring.Handler{Store: s.store, Uploads: s.uploads, UploadsDir: s.cfg.UploadsDir}.Routes(r)
				checks.Handler{Cfg: s.cfg, Backups: s.backups, Mail: s.mail.Status}.Routes(r)
			})

			r.Group(func(r chi.Router) {
				r.Use(s.guard.RequireAdmin)
				r.Use(s.guard.SessionOnly)
				settingsH.AdminSessionRoutes(r)
				oauthH.AdminRoutes(r)
				mailing.Handler{Mail: s.mail}.Routes(r)
				spaces.AdminSessionRoutes(r)
				backupsH.AdminRoutes(r)
				transferH.AdminSessionRoutes(r)
			})

			settingsH.AccountRoutes(r)
		})
	})

	// MCP clients (Claude web) discover and sign in through these; they need
	// the public URL because it is their issuer.
	r.Group(func(r chi.Router) {
		r.Use(s.guard.NeedAppURL)
		oauthH.Routes(r)
	})

	r.Get("/*", s.spa.ServeHTTP)
	return r
}

// spaceRoutes are everything under /api/spaces/{space}.
func (s *Server) spaceRoutes(r chi.Router) {
	r.Use(s.guard.WithSpace)
	r.Use(s.guard.RequireTokenGrant)
	s.spaceHandler().ReadRoutes(r)
	s.settingsHandler().SpaceReadRoutes(r)
	s.transferHandler().ReadRoutes(r)
	s.syncGroupsHandler().ReadRoutes(r)

	// Administering the space: its admins, in a real session.
	r.Group(func(r chi.Router) {
		r.Use(s.guard.RequireSpaceAdmin)
		r.Use(s.guard.SessionOnly)
		s.spaceHandler().AdminRoutes(r)
		s.settingsHandler().SpaceAdminRoutes(r)
		s.transferHandler().AdminRoutes(r)
		s.backupsHandler().SpaceRoutes(r)
	})

	r.Group(func(r chi.Router) {
		r.Use(s.guard.RequireWrite)
		dataRoutes(s, r)
		s.transferHandler().WriteRoutes(r)
	})
}

// dataRoutes are the finances of a space, under /api/spaces/{space}.
func dataRoutes(s *Server, r chi.Router) {
	currencies.Routes(r)
	accounts.Handler{Loc: s.cfg.Location}.Routes(r)
	categories.Handler{Refsync: s.refsync}.Routes(r)
	tags.Handler{Refsync: s.refsync}.Routes(r)
	transactions.Routes(r)
	imports.Handler{Uploads: s.uploads, Queue: s.queue}.Routes(r)
	debts.Routes(r)
	reports.Routes(r)
	recurring.Routes(r)
	budgets.Routes(r)
	automation.Routes(r)
	uploadapi.Handler{Uploads: s.uploads}.Routes(r)
}

func (s *Server) backupsHandler() backups.Handler {
	return backups.Handler{
		Cfg: s.cfg, Store: s.store, Backups: s.backups, Spaces: s.spaces, Settings: s.settings,
		Users: s.users, Bootstrap: s.bootstrap, Guard: s.guard,
	}
}

func (s *Server) usersHandler() users.Handler {
	return users.Handler{Cfg: s.cfg, Users: s.users, Tokens: s.tokens, Spaces: s.spaces, Mail: s.mail, Backups: s.backups, Login: s.login}
}

func (s *Server) spaceHandler() spaceapi.Handler {
	return spaceapi.Handler{
		Cfg: s.cfg, Store: s.store, Spaces: s.spaces, Users: s.users, Backups: s.backups, Mail: s.mail,
		Login: s.login, Verify: s.verify, Creator: s.spaceCreator(), Guard: s.guard,
	}
}

func (s *Server) spaceCreator() spacefill.Creator {
	return spacefill.Creator{Spaces: s.spaces, Store: s.store, Rates: s.rates}
}

func (s *Server) settingsHandler() settingsapi.Handler {
	return settingsapi.Handler{Cfg: s.cfg, Settings: s.settings, SSO: &s.sso, Verify: s.verify}
}

func (s *Server) transferHandler() transferapi.Handler {
	return transferapi.Handler{Cfg: s.cfg, Spaces: s.spaces, Transfers: s.transfers, Keys: s.keys}
}

func (s *Server) syncGroupsHandler() syncgroups.Handler {
	return syncgroups.Handler{Spaces: s.spaces, Refsync: s.refsync, Guard: s.guard}
}
