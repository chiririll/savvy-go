package httpserver

// One handler package per resource of the space API (accounts to debts).
import (
	_ "savvy-go/internal/httpserver/api/accounts"
	_ "savvy-go/internal/httpserver/api/apitokens"
	_ "savvy-go/internal/httpserver/api/automation"
	_ "savvy-go/internal/httpserver/api/backups"
	_ "savvy-go/internal/httpserver/api/budgets"
	_ "savvy-go/internal/httpserver/api/catalog"
	_ "savvy-go/internal/httpserver/api/categories"
	_ "savvy-go/internal/httpserver/api/currencies"
	_ "savvy-go/internal/httpserver/api/debts"
)
