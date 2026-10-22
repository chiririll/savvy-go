package httpserver

import (
	"savvy-go/internal/appconfig"
	"savvy-go/internal/automation"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/importer"
	"savvy-go/internal/ledger"
	"savvy-go/internal/locator"
	"savvy-go/internal/recurring"
	"savvy-go/internal/reports"
	"savvy-go/internal/settings"
	"savvy-go/internal/store"
	"savvy-go/internal/uploads"
)

func (s *Server) newScope(id int64, role string, d store.DB) *httpx.SpaceScope {
	txs := ledger.Transactions{DB: d}
	accounts := ledger.Accounts{DB: d}
	return &httpx.SpaceScope{
		ID:         id,
		Role:       role,
		DB:         d,
		Settings:   settings.Store{DB: d, Space: true},
		Currencies: ledger.Currencies{DB: d, Source: locator.Get[ledger.RateSource](s.loc)},
		Accounts:   accounts,
		Categories: ledger.Categories{DB: d},
		Tags:       ledger.Tags{DB: d},
		Txs:        txs,
		Debts:      ledger.Debts{Accounts: accounts, Transactions: txs},
		Recurring:  recurring.Service{DB: d, Txs: txs},
		Budgets:    ledger.Budgets{DB: d},
		Automation: automation.Service{DB: d, Txs: txs},
		Reports:    reports.Service{DB: d, Loc: locator.Get[appconfig.Config](s.loc).Location},
		Imports:    importer.Service{DB: d, Uploads: locator.Get[uploads.Service](s.loc), Txs: txs},
	}
}
