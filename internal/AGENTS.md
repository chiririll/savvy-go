# Backend (Go)

```bash
go run ./cmd/savvy-go       # serves :8080, data in ./savvy-data
```

Nested files: `db/` (SQL, sqlc), `httpserver/` (routes, tokens, OAuth, MCP), `migrate/`, `legacy/` (Laravel).

## Store boundary

- Business packages reach storage only via `internal/store`. No import of `store/sqlite`, `migrate`, `legacy`, `database/sql`, `db/sqlc`. No `PRAGMA`, `ATTACH`, `sqlite_master`. `store/arch_test.go` enforces. Reason: space DB may become Postgres schema.
- Work touching two spaces: `store.InSpaces`.

## Packages

- Dependencies point down. `store/arch_test.go` enforces layering. Core never imports `net/http`.
- `currencydata` only client of currency service. `ledger` sees only `ledger.RateSource`. Tests: `currencydata.NewOffline()`.
- `transfers` uses `spaces`, `signing`, not `ledger`.
- `sso`: new user gets no space. Never assume user has space.
- Shared test fixtures cannot live in `ledger` (import cycle). `reports` keeps own copy.

## Money

- `internal/money`: exact integer minor units plus currency. No float. No adding across currencies.
- Bounded by `MaxMinor`: SUM cannot overflow, JSON numbers stay exact for JS.
