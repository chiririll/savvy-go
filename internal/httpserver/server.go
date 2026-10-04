package httpserver

import (
	"net/http"
	"os"
	"strings"

	"savvy-go/internal/auth"
	"savvy-go/internal/config"
	"savvy-go/internal/domain"
	"savvy-go/internal/jobs"
	"savvy-go/internal/settings"
	"savvy-go/internal/signing"
	"savvy-go/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server is the HTTP front door: health probes, /api, and the Vite SPA.
// Server-level services use the server database; the services of a space are
// built per request by withSpace (see spaceScope).
type Server struct {
	cfg        config.Config
	store      store.Store
	mux        *chi.Mux
	users      auth.Users
	sessions   auth.Sessions
	apiTokens  auth.APITokens
	tokens     auth.PasswordTokens
	challenges auth.Challenges
	settings   settings.Store
	spaces     domain.Spaces
	backups    domain.Backups
	uploads    domain.Uploads
	sso        domain.SSO
	twoFactor  auth.TwoFactor
	webauthn   auth.WebAuthn
	queue      *jobs.Queue
}

func New(cfg config.Config, st store.Store, keys *signing.Holder) *Server {
	domain.SetLocation(cfg.Location)
	srv := st.Server()
	s := &Server{
		cfg:        cfg,
		store:      st,
		users:      auth.Users{DB: srv},
		sessions:   auth.Sessions{DB: srv, Cfg: cfg},
		apiTokens:  auth.APITokens{DB: srv},
		tokens:     auth.PasswordTokens{DB: srv},
		challenges: auth.Challenges{DB: srv, Cfg: cfg},
		settings:   settings.Store{DB: srv},
		spaces:     domain.Spaces{Store: st},
		uploads:    domain.Uploads{DB: srv, Root: cfg.UploadsDir, AppURL: cfg.AppURL, SignSecret: cfg.AppURL + "|upload"},
		twoFactor:  auth.TwoFactor{DB: srv, Users: auth.Users{DB: srv}, AppKey: cfg.AppKey},
		webauthn:   auth.WebAuthn{DB: srv, Cfg: cfg},
	}
	s.backups = domain.Backups{Store: st, Keys: domain.KeyRing{Holder: keys}, Dir: cfg.BackupsDir, DataDir: cfg.DataDir}
	s.sso = domain.SSO{DB: srv, Users: s.users, Settings: s.settings, Spaces: s.spaces, AppURL: cfg.AppURL}
	_ = os.MkdirAll(cfg.UploadsDir, 0o775)
	_ = os.MkdirAll(cfg.BackupsDir, 0o775)
	s.mux = s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(locale)
	r.Use(noStoreAPI)

	r.Get("/livez", s.livez)
	r.Get("/readyz", s.readyz)

	r.Route("/api", func(r chi.Router) {
		r.Get("/docs", s.apiDocs)
		r.Get("/openapi.yaml", s.apiSpec)

		r.Get("/auth/status", s.authStatus)
		r.Get("/auth/me", s.authMe)
		r.Post("/auth/register", s.authRegister)
		r.Post("/auth/login", s.authLogin)
		r.Get("/auth/password/{token}", s.passwordPreview)
		r.Post("/auth/password/{token}", s.passwordAccept)
		r.Post("/auth/2fa/verify", s.twoFactorVerify)
		r.Get("/auth/sso/providers", s.ssoProviders)
		r.Post("/auth/sso/exchange", s.ssoExchange)
		r.Get("/auth/sso/{slug}/redirect", s.ssoRedirect)
		r.Get("/auth/sso/{slug}/callback", s.ssoCallback)
		r.Post("/auth/sso/{slug}/acs", s.ssoACS)
		r.Get("/auth/sso/{slug}/metadata", s.ssoMetadata)
		r.Post("/auth/webauthn/login/options", s.webauthnLoginOptions)
		r.Post("/auth/webauthn/login/verify", s.webauthnLoginVerify)
		r.Put("/uploads/{id}/parts/{part}", s.uploadPart)
		r.Get("/invitations/{token}", s.invitationPreview)
		r.Post("/invitations/{token}/register", s.invitationRegister)

		r.Group(func(r chi.Router) {
			r.Use(s.requireSession)
			r.Use(s.requireCSRF)

			// Credential management is limited to real sessions, never API tokens.
			r.Group(func(r chi.Router) {
				r.Use(s.sessionOnly)
				r.Post("/auth/logout", s.authLogout)
				r.Post("/auth/logout-others", s.authLogoutOthers)
				r.Put("/auth/password", s.authChangePassword)
				r.Get("/auth/2fa/status", s.twoFactorStatus)
				r.Get("/auth/webauthn/credentials", s.webauthnIndex)
				r.Get("/auth/api-tokens", s.apiTokensIndex)
				r.Post("/auth/api-tokens", s.apiTokensStore)
				r.Delete("/auth/api-tokens/{id}", s.apiTokensDestroy)
				r.Post("/auth/2fa/enable", s.twoFactorEnable)
				r.Post("/auth/2fa/confirm", s.twoFactorConfirm)
				r.Post("/auth/2fa/disable", s.twoFactorDisable)
				r.Get("/auth/2fa/recovery-codes", s.twoFactorRecoveryCodes)
				r.Post("/auth/2fa/recovery-codes/regenerate", s.twoFactorRegenerate)
				r.Post("/auth/webauthn/register/options", s.webauthnRegisterOptions)
				r.Post("/auth/webauthn/register/verify", s.webauthnRegisterVerify)
				r.Patch("/auth/webauthn/credentials/{id}", s.webauthnUpdate)
				r.Delete("/auth/webauthn/credentials/{id}", s.webauthnDestroy)
			})

			r.Get("/spaces", s.spacesIndex)
			r.With(s.sessionOnly).Post("/spaces", s.spacesStore)
			r.With(s.sessionOnly).Post("/spaces/import", s.spacesImport)
			r.With(s.sessionOnly).Post("/invitations/{token}/accept", s.invitationAccept)
			r.Route("/spaces/{space}", func(r chi.Router) {
				r.Use(s.withSpace)
				r.Get("/", s.spaceShow)
				r.Get("/members", s.membersIndex)
				r.Get("/settings", s.spaceSettingsIndex)
				r.With(s.sessionOnly).Post("/leave", s.spaceLeave)

				// Administering the space: its admins, in a real session.
				r.Group(func(r chi.Router) {
					r.Use(s.requireSpaceAdmin)
					r.Use(s.sessionOnly)
					r.Patch("/", s.spaceUpdate)
					r.Delete("/", s.spaceDestroy)
					r.Patch("/members/{user}", s.membersUpdate)
					r.Delete("/members/{user}", s.membersDestroy)
					r.Get("/invitations", s.invitationsIndex)
					r.Post("/invitations", s.invitationsStore)
					r.Delete("/invitations/{id}", s.invitationsDestroy)
					r.Patch("/settings", s.spaceSettingsUpdate)
					r.Get("/audit", s.spaceAudit)
					// P1: a space's backups are for its admins, never for API tokens.
					r.Get("/backups", s.spaceBackupsIndex)
					r.Post("/backups", s.spaceBackupsStore)
					r.Post("/backups/upload", s.spaceBackupsUpload)
					r.Get("/backups/{name}/download", s.spaceBackupsDownload)
					r.Post("/backups/{name}/restore", s.spaceBackupsRestore)
					r.Delete("/backups/{name}", s.spaceBackupsDestroy)
				})

				r.Group(func(r chi.Router) {
					r.Use(s.requireWrite)
					dataRoutes(s, r)
				})
			})

			// P2: users, instance settings, monitoring and the space overview
			// are the server admins'.
			r.Group(func(r chi.Router) {
				r.Use(s.requireAdmin)
				r.Get("/users", s.usersIndex)
				r.Get("/users/{id}", s.usersShow)
				r.Post("/users", s.usersStore)
				r.Post("/users/{id}/password-token", s.usersIssueToken)
				r.Put("/users/{id}", s.usersUpdate)
				r.Patch("/users/{id}", s.usersUpdate)
				r.Delete("/users/{id}", s.usersDestroy)
				r.Get("/auth/sso/presets", s.ssoPresets)
				r.Get("/identity-providers", s.idpIndex)
				r.Post("/identity-providers", s.idpStore)
				r.Get("/identity-providers/{id}", s.idpShow)
				r.Put("/identity-providers/{id}", s.idpUpdate)
				r.Patch("/identity-providers/{id}", s.idpUpdate)
				r.Delete("/identity-providers/{id}", s.idpDestroy)
				r.Post("/identity-providers/{id}/test", s.idpTest)
				r.Get("/monitoring/storage", s.monitoringStorage)
				r.Get("/monitoring/resources", s.monitoringResources)
			})

			r.Group(func(r chi.Router) {
				r.Use(s.requireAdmin)
				r.Use(s.sessionOnly)
				r.Patch("/settings", s.settingsUpdate)
				// P1: backups hold every space.
				r.Get("/backups", s.backupsIndex)
				r.Post("/backups", s.backupsStore)
				r.Post("/backups/upload", s.backupsUpload)
				r.Get("/backups/{name}/download", s.backupsDownload)
				r.Post("/backups/{name}/restore", s.backupsRestore)
				r.Delete("/backups/{name}", s.backupsDestroy)
				r.Get("/admin/spaces", s.adminSpacesIndex)
				r.Patch("/admin/spaces/{id}", s.adminSpaceUpdate)
				r.Delete("/admin/spaces/{id}", s.adminSpaceDestroy)
				r.Post("/admin/spaces/{id}/admins", s.adminSpaceAssignAdmin)
				r.Get("/admin/deleted-spaces", s.adminDeletedIndex)
				r.Post("/admin/deleted-spaces/{name}/restore", s.adminDeletedRestore)
			})

			// Instance settings everyone may read (sign-in options).
			r.Get("/settings", s.settingsIndex)
		})
	})

	r.Get("/*", s.spa)
	return r
}

func locale(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if loc := strings.TrimSpace(r.Header.Get("X-Locale")); loc != "" {
			w.Header().Set("Content-Language", loc)
		}
		next.ServeHTTP(w, r)
	})
}

func noStoreAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/livez" || r.URL.Path == "/readyz" {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

// dataRoutes are the finances of a space, under /api/spaces/{space}.
func dataRoutes(s *Server, r chi.Router) {
	r.Get("/currencies/catalog", s.currenciesCatalog)
	r.Get("/currencies", s.currenciesIndex)
	r.Post("/currencies", s.currenciesStore)
	r.Get("/currencies/{id}", s.currenciesShow)
	r.Put("/currencies/{id}", s.currenciesUpdate)
	r.Patch("/currencies/{id}", s.currenciesUpdate)
	r.Delete("/currencies/{id}", s.currenciesDestroy)
	r.Post("/currencies/{id}/set-base", s.currenciesSetBase)
	r.Post("/currencies/convert", s.currenciesConvert)

	r.Post("/accounts/reorder", s.accountsReorder)
	r.Get("/accounts", s.accountsIndex)
	r.Post("/accounts", s.accountsStore)
	r.Get("/accounts/{id}", s.accountsShow)
	r.Put("/accounts/{id}", s.accountsUpdate)
	r.Patch("/accounts/{id}", s.accountsUpdate)
	r.Delete("/accounts/{id}", s.accountsDestroy)
	r.Get("/accounts-balance-history", s.accountsBalanceHistory)
	r.Get("/accounts-balance-comparison", s.accountsBalanceComparison)

	r.Get("/categories", s.categoriesIndex)
	r.Post("/categories", s.categoriesStore)
	r.Get("/categories/{id}", s.categoriesShow)
	r.Put("/categories/{id}", s.categoriesUpdate)
	r.Patch("/categories/{id}", s.categoriesUpdate)
	r.Delete("/categories/{id}", s.categoriesDestroy)
	r.Post("/categories/{id}/set-default", s.categoriesSetDefault)
	r.Get("/categories/{id}/statistics", s.categoriesStatistics)
	r.Get("/categories-summary", s.categoriesSummary)

	r.Get("/tags", s.tagsIndex)
	r.Post("/tags", s.tagsStore)
	r.Get("/tags/{id}", s.tagsShow)
	r.Put("/tags/{id}", s.tagsUpdate)
	r.Patch("/tags/{id}", s.tagsUpdate)
	r.Delete("/tags/{id}", s.tagsDestroy)

	r.Get("/transactions", s.transactionsIndex)
	r.Post("/transactions", s.transactionsStore)
	r.Get("/transactions/{id}", s.transactionsShow)
	r.Put("/transactions/{id}", s.transactionsUpdate)
	r.Patch("/transactions/{id}", s.transactionsUpdate)
	r.Delete("/transactions/{id}", s.transactionsDestroy)
	r.Post("/transactions/{id}/duplicate", s.transactionsDuplicate)
	r.Post("/transactions/{id}/confirm", s.transactionsConfirm)
	r.Post("/transactions/{id}/skip", s.transactionsSkip)
	r.Get("/transactions-summary", s.transactionsSummary)
	r.Get("/transactions-pending-summary", s.transactionsPendingSummary)

	r.Get("/debts", s.debtsIndex)
	r.Post("/debts", s.debtsStore)
	r.Get("/debts/{id}", s.debtsShow)
	r.Put("/debts/{id}", s.debtsUpdate)
	r.Patch("/debts/{id}", s.debtsUpdate)
	r.Delete("/debts/{id}", s.debtsDestroy)
	r.Post("/debts/{id}/payment", s.debtsPayment)
	r.Post("/debts/{id}/collect", s.debtsCollect)
	r.Post("/debts/{id}/reopen", s.debtsReopen)
	r.Get("/debts-summary", s.debtsSummary)

	r.Get("/reports/overview", s.reportsOverview)
	r.Get("/reports/money-flow", s.reportsMoneyFlow)
	r.Get("/reports/expense-pace", s.reportsExpensePace)
	r.Get("/reports/expenses-by-category", s.reportsByCategory)
	r.Get("/reports/cash-flow-over-time", s.reportsCashFlow)
	r.Get("/reports/activity-heatmap", s.reportsHeatmap)
	r.Get("/reports/transactions/summary", s.reportsTxSummary)
	r.Get("/reports/transactions/by-category", s.reportsTxByCategory)
	r.Get("/reports/transactions/dynamics", s.reportsTxDynamics)
	r.Get("/reports/transactions/top", s.reportsTxTop)
	r.Get("/reports/net-worth", s.reportsNetWorth)
	r.Get("/reports/net-worth-history", s.reportsNetWorthHistory)

	r.Get("/recurring", s.recurringIndex)
	r.Get("/recurring-upcoming", s.recurringUpcoming)
	r.Post("/recurring", s.recurringStore)
	r.Get("/recurring/{id}", s.recurringShow)
	r.Put("/recurring/{id}", s.recurringUpdate)
	r.Patch("/recurring/{id}", s.recurringUpdate)
	r.Delete("/recurring/{id}", s.recurringDestroy)

	r.Get("/budgets", s.budgetsIndex)
	r.Post("/budgets", s.budgetsStore)
	r.Get("/budgets/{id}", s.budgetsShow)
	r.Put("/budgets/{id}", s.budgetsUpdate)
	r.Patch("/budgets/{id}", s.budgetsUpdate)
	r.Delete("/budgets/{id}", s.budgetsDestroy)

	r.Get("/automation-rules/triggers", s.automationTriggers)
	r.Post("/automation-rules/reorder", s.automationReorder)
	r.Get("/automation-rules", s.automationIndex)
	r.Post("/automation-rules", s.automationStore)
	r.Get("/automation-rules/{id}", s.automationShow)
	r.Put("/automation-rules/{id}", s.automationUpdate)
	r.Patch("/automation-rules/{id}", s.automationUpdate)
	r.Delete("/automation-rules/{id}", s.automationDestroy)
	r.Post("/automation-rules/{id}/toggle", s.automationToggle)
	r.Post("/automation-rules/{id}/test", s.automationTest)
	r.Get("/automation-rules/{id}/logs", s.automationLogs)
	r.Post("/s3/multipart", s.uploadCreate)
	r.Get("/s3/multipart/{upload}", s.uploadListParts)
	r.Get("/s3/multipart/{upload}/{part}", s.uploadSignPart)
	r.Post("/s3/multipart/{upload}/complete", s.uploadComplete)
	r.Delete("/s3/multipart/{upload}", s.uploadAbort)

	r.Post("/transactions/import/parse", s.importParse)
	r.Post("/transactions/import/preview", s.importPreview)
	r.Post("/transactions/import/execute", s.importExecute)
	r.Get("/transactions/import/{import}", s.importShow)
}
