package httpserver

import (
	"savvy-go/internal/automation"
	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/guard"
	"savvy-go/internal/importer"
	"savvy-go/internal/ledger"
	"savvy-go/internal/recurring"
	"savvy-go/internal/reports"
	"savvy-go/internal/settings"
	"savvy-go/internal/store"
)

func (s *Server) newScope(id int64, role string, d store.DB) *api.SpaceScope {
	txs := ledger.Transactions{DB: d}
	accounts := ledger.Accounts{DB: d}
	return &api.SpaceScope{
		ID:         id,
		Role:       role,
		DB:         d,
		Settings:   settings.Store{DB: d, Space: true},
		Currencies: ledger.Currencies{DB: d, Source: s.rates},
		Accounts:   accounts,
		Categories: ledger.Categories{DB: d},
		Tags:       ledger.Tags{DB: d},
		Txs:        txs,
		Debts:      ledger.Debts{Accounts: accounts, Transactions: txs},
		Recurring:  recurring.Service{DB: d, Txs: txs},
		Budgets:    ledger.Budgets{DB: d},
		Automation: automation.Service{DB: d, Txs: txs},
		Reports:    reports.Service{DB: d, Loc: s.cfg.Location},
		Imports:    importer.Service{DB: d, Uploads: s.uploads, Txs: txs},
	}
}

func (s *Server) newGuard() *guard.Guard {
	return &guard.Guard{
		Cfg:           s.cfg,
		Sessions:      s.sessions,
		APITokens:     s.apiTokens,
		EmailRequired: s.login.EmailRequired,
		Space: guard.SpaceDeps{
			Spaces: s.spaces, Store: s.store, Refsync: s.refsync, NewScope: s.newScope,
		},
	}
}
