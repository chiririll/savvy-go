package httpserver

import (
	"database/sql"
	"net/http"
	"os"
	"strings"

	"github.com/chiririll/savvy-plus/internal/auth"
	"github.com/chiririll/savvy-plus/internal/config"
	"github.com/chiririll/savvy-plus/internal/domain"
	"github.com/chiririll/savvy-plus/internal/jobs"
	"github.com/chiririll/savvy-plus/internal/settings"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server is the HTTP front door: health probes, /api, and the Vite SPA.
type Server struct {
	cfg        config.Config
	db         *sql.DB
	mux        *chi.Mux
	users      auth.Users
	sessions   auth.Sessions
	tokens     auth.PasswordTokens
	challenges auth.Challenges
	settings   settings.Store
	currencies domain.Currencies
	accounts   domain.Accounts
	categories domain.Categories
	tags       domain.Tags
	txs        domain.Transactions
	debts      domain.Debts
	recurring  domain.RecurringStore
	budgets    domain.Budgets
	automation domain.Automation
	reports    domain.Reports
	uploads    domain.Uploads
	imports    domain.Imports
	backups    domain.Backups
	sso        domain.SSO
	twoFactor  auth.TwoFactor
	webauthn   auth.WebAuthn
	queue      *jobs.Queue
}

func New(cfg config.Config, sqlDB *sql.DB) *Server {
	s := &Server{
		cfg:        cfg,
		db:         sqlDB,
		users:      auth.Users{DB: sqlDB},
		sessions:   auth.Sessions{DB: sqlDB, Cfg: cfg},
		tokens:     auth.PasswordTokens{DB: sqlDB},
		challenges: auth.Challenges{DB: sqlDB, Cfg: cfg},
		settings:   settings.Store{DB: sqlDB},
		currencies: domain.Currencies{DB: sqlDB},
		accounts:   domain.Accounts{DB: sqlDB},
		categories: domain.Categories{DB: sqlDB},
		tags:       domain.Tags{DB: sqlDB},
		txs:        domain.Transactions{DB: sqlDB},
		debts:      domain.Debts{Accounts: domain.Accounts{DB: sqlDB}, Transactions: domain.Transactions{DB: sqlDB}},
		recurring:  domain.RecurringStore{DB: sqlDB, Txs: domain.Transactions{DB: sqlDB}},
		budgets:    domain.Budgets{DB: sqlDB},
		automation: domain.Automation{DB: sqlDB, Txs: domain.Transactions{DB: sqlDB}},
		reports:    domain.Reports{DB: sqlDB, Loc: cfg.Location},
		uploads:    domain.Uploads{DB: sqlDB, Root: cfg.UploadsDir, AppURL: cfg.AppURL, SignSecret: cfg.AppURL + "|upload"},
		backups:    domain.Backups{DB: sqlDB, Dir: cfg.BackupsDir, Database: cfg.Database},
		twoFactor:  auth.TwoFactor{DB: sqlDB, Users: auth.Users{DB: sqlDB}},
		webauthn:   auth.WebAuthn{DB: sqlDB, Cfg: cfg},
	}
	s.sso = domain.SSO{DB: sqlDB, Users: s.users, Settings: s.settings, AppURL: cfg.AppURL}
	s.imports = domain.Imports{DB: sqlDB, Uploads: s.uploads, Txs: s.txs}
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

		r.Group(func(r chi.Router) {
			r.Use(s.requireSession)
			r.Use(s.requireCSRF)

			r.Post("/auth/logout", s.authLogout)
			r.Post("/auth/logout-others", s.authLogoutOthers)
			r.Put("/auth/password", s.authChangePassword)
			r.Get("/auth/2fa/status", s.twoFactorStatus)
			r.Get("/auth/webauthn/credentials", s.webauthnIndex)

			r.Get("/users", s.usersIndex)
			r.Get("/users/{id}", s.usersShow)
			r.Group(func(r chi.Router) {
				r.Use(s.requireAdmin)
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
			})

			r.Group(func(r chi.Router) {
				r.Use(s.requireWrite)
				r.Get("/settings", s.settingsIndex)
				r.Patch("/settings", s.settingsUpdate)

				r.Post("/auth/2fa/enable", s.twoFactorEnable)
				r.Post("/auth/2fa/confirm", s.twoFactorConfirm)
				r.Post("/auth/2fa/disable", s.twoFactorDisable)
				r.Get("/auth/2fa/recovery-codes", s.twoFactorRecoveryCodes)
				r.Post("/auth/2fa/recovery-codes/regenerate", s.twoFactorRegenerate)
				r.Post("/auth/webauthn/register/options", s.webauthnRegisterOptions)
				r.Post("/auth/webauthn/register/verify", s.webauthnRegisterVerify)
				r.Patch("/auth/webauthn/credentials/{id}", s.webauthnUpdate)
				r.Delete("/auth/webauthn/credentials/{id}", s.webauthnDestroy)

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
				r.Get("/monitoring/storage", s.monitoringStorage)
				r.Get("/monitoring/resources", s.monitoringResources)

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

				r.Get("/backups", s.backupsIndex)
				r.Post("/backups", s.backupsStore)
				r.Post("/backups/upload", s.backupsUpload)
				r.Get("/backups/{id}/download", s.backupsDownload)
				r.Get("/backups/{id}/inspect", s.backupsInspect)
				r.Post("/backups/{id}/restore", s.backupsRestore)
				r.Delete("/backups/{id}", s.backupsDestroy)
			})
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
