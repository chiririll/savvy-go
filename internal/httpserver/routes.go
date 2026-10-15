package httpserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (s *Server) routes() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(locale)
	r.Use(noStoreAPI)
	r.Use(noFraming)

	r.Get("/livez", s.livez)
	r.Get("/readyz", s.readyz)

	r.Route("/api", func(r chi.Router) {
		r.Get("/docs", s.apiDocs)
		r.Get("/openapi.yaml", s.apiSpec)

		s.publicAuthRoutes(r)
		s.publicOnboardingRestoreRoutes(r)
		s.publicEmailVerificationRoutes(r)
		s.publicTwofactorRoutes(r)
		s.publicSsoRoutes(r)
		s.publicWebauthnRoutes(r)
		s.publicUploadsRoutes(r)
		s.publicSpacesApiRoutes(r)

		r.Group(func(r chi.Router) {
			r.Use(s.requireSession)
			r.Use(s.requireCSRF)
			r.Use(s.requireVerifiedEmail)

			// Credential management is limited to real sessions, never API tokens.
			r.Group(func(r chi.Router) {
				r.Use(s.sessionOnly)
				s.credentialAuthRoutes(r)
				s.credentialEmailVerificationRoutes(r)
				s.credentialTwofactorRoutes(r)
				s.credentialWebauthnRoutes(r)
				s.credentialApiTokensRoutes(r)
				s.credentialOauthRoutes(r)
			})

			s.accountCatalogRoutes(r)
			s.accountSpacesApiRoutes(r)
			s.accountSpaceRoutes(r)
			s.accountBackupsRoutes(r)
			s.accountSyncGroupsRoutes(r)
			r.Route("/spaces/{space}", s.spaceRoutes)

			// P2: users, instance settings, monitoring and the space overview
			// are the server admins'.
			r.Group(func(r chi.Router) {
				r.Use(s.requireAdmin)
				s.adminUsersRoutes(r)
				s.adminSsoRoutes(r)
				s.adminMonitoringRoutes(r)
				s.adminChecksRoutes(r)
			})

			r.Group(func(r chi.Router) {
				r.Use(s.requireAdmin)
				r.Use(s.sessionOnly)
				s.adminSessionSettingsRoutes(r)
				s.adminSessionOauthRoutes(r)
				s.adminSessionMailRoutes(r)
				s.adminSessionSpacesApiRoutes(r)
				s.adminSessionBackupsRoutes(r)
				s.adminSessionTransfersRoutes(r)
			})

			s.accountSettingsRoutes(r)
		})
	})

	// MCP clients (Claude web) discover and sign in through these; they need
	// the public URL because it is their issuer.
	r.Group(func(r chi.Router) {
		r.Use(s.needAppURL)
		s.mcpRoutes(r)
	})

	r.Get("/*", s.spa)
	return r
}

// spaceRoutes are everything under /api/spaces/{space}.
func (s *Server) spaceRoutes(r chi.Router) {
	r.Use(s.withSpace)
	r.Use(s.requireTokenGrant)
	s.spaceReadSpacesApiRoutes(r)
	s.spaceReadTransfersRoutes(r)
	s.spaceReadSyncGroupsRoutes(r)

	// Administering the space: its admins, in a real session.
	r.Group(func(r chi.Router) {
		r.Use(s.requireSpaceAdmin)
		r.Use(s.sessionOnly)
		s.spaceAdminSpacesApiRoutes(r)
		s.spaceAdminSettingsRoutes(r)
		s.spaceAdminTransfersRoutes(r)
		s.spaceAdminBackupsRoutes(r)
	})

	r.Group(func(r chi.Router) {
		r.Use(s.requireWrite)
		dataRoutes(s, r)
		s.spaceWriteTransfersRoutes(r)
	})
}

// dataRoutes are the finances of a space, under /api/spaces/{space}.
func dataRoutes(s *Server, r chi.Router) {
	s.currencyRoutes(r)
	s.accountRoutes(r)
	s.categoryRoutes(r)
	s.tagRoutes(r)
	s.transactionRoutes(r)
	s.importRoutes(r)
	s.debtRoutes(r)
	s.reportRoutes(r)
	s.recurringRoutes(r)
	s.budgetRoutes(r)
	s.automationRoutes(r)
	s.uploadRoutes(r)
}
