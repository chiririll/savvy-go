# Backend (Go)

```bash
go generate ./internal/db   # sqlc: queries -> internal/db/sqlc
go run ./cmd/savvy-go       # serves :8080, data in ./savvy-data
```

## Store boundary

- `ledger`, `spaces`, `transfers`, `uploads`, `automation`, `recurring`, `reports`, `calendar`, `currencydata`, `sso`, `backup`, `importer`, `legacy/restore`, `httpserver`, `auth`, `settings`, `seed`, `signing`, `schedule`, `jobs` use storage only via `internal/store`. No `store/sqlite`, `migrate`, `legacy`, `modernc.org/sqlite`, `database/sql`, `db/sqlc` imports. No `PRAGMA`, `ATTACH`, `sqlite_master` in SQL. `store/arch_test.go` enforces. Reason: space DB may become Postgres schema later.
- Work touching two spaces: `store.InSpaces`.

## Packages

Dependencies point down. Each package may import the ones above it in this list, never below.

- `calendar`, `money`, `signing`, `db`, `store`: leaves.
- `currencydata`: the one place that talks to the currency data service (names, rates). One `Source` per server, shared by all spaces. Update rates are cached in a map by base currency (spaces with the same base share a request; the job calls `Expire` once per run); catalog look-ups fetch live and skip that cache. `ledger` only sees the `ledger.RateSource` interface. Tests use `currencydata.NewOffline()`.
- `spaces`: registry, members, roles, invitations, quotas, audit log. Needs from outside go through small interfaces (`spaces.FinalBackuper`).
- `ledger`: finance core of one space: accounts, transactions, categories, tags, currencies, budgets, debts. Knows nothing of the packages below.
- On top of `ledger`, independent of each other: `reports`, `recurring`, `automation`, `importer` (CSV; also uses `uploads`), `uploads`.
- `transfers`: signed transfers between linked spaces. Uses `spaces`, `signing`; not `ledger`.
- `backup`: backup files, archive format, restore. Uses `spaces`, `transfers`, `signing`.
- `sso`: identity providers, OIDC/OAuth/SAML, JIT provisioning. `ProvisionSpace` is injected.
- `store/arch_test.go` enforces this layering and that the core never imports `net/http`.
- Shared test fixtures cannot live in `ledger` (its in-package tests would import-cycle), so `reports` keeps its own small copy.

## SQL

- SQL lives only in the database packages: `internal/db` (+ `sqlc`, `filter`), the repositories `internal/db/repo/<area>`, `store/sqlite`, `migrate`, `legacy`. Business code never builds a query or sees a `sql.Null*`.
- A repository (`internal/db/repo/<area>`: `auth`, `ledger`, `spaces`, `sso`, `transfers`, `recurring`, `automation`, `importer`, `settings`, `uploads`; imported as `repo` by its own package) takes a `store.DB` and speaks plain Go: `time.Time`, pointers for optional values, `money.Money` for amounts, nil for "no such row", a bool for "did it change a row". It decides nothing: rules and the clock (it is handed `now`) stay in the business package. It never imports a business package (tested).
- A package imports only its own repository (as `repo`). Rows of another area come through that area's package. Exceptions: `reports` and `seed` (tested in `store/arch_test.go`). Repo tests may import other repos for fixtures.
- `Repo{DB: tx}` works inside `store.Tx`; build it from the transaction handle.
- New query: add the `.sql`, run generate, add a repository method, call that. Test the method in `repo/<area>` with `db/dbtest` (real migrated DB). Tests may still use raw SQL to set up or check stored rows.
- Queries in `internal/db/queries/{server,space}`. Server queries touch only server tables, space only space tables. Keep SQL Postgres-compatible. Tests enforce.
- `internal/db/sqlc` generated, never edit. Change query, run `go generate ./internal/db`.
- sqlc sqlite: use each `narg` once per query (COALESCE/CASE), never `OR col = ?N`. Driver counts every `?`.
- Dynamic SQL (IN lists, report GROUP BY) only in `internal/db/filter`.

## Money

- `internal/money`: exact integer minor units plus currency. No float, no adding across currencies.
- Amounts bounded by `MaxMinor`: keeps SUM from overflowing, JSON numbers exact for JS.
- Test data: `money/moneytest`.

## Migrations

- `internal/migrate/sql/{server,space}/0001_initial.sql`. Edit in place.
- `migrate`: upgrades between Go versions, not released yet.
- `legacy`: loads Laravel-era DB once, on a staging copy. Rarely touched. No new Go-era logic.
- `legacy/restore`: Laravel upgrade entry points. Server and space: convert, then save only the upgraded file as an ordinary backup (`backup.SaveServerFrom/SaveSpaceFrom`); live data is touched only when the user restores it. The Laravel file is never kept. New space: created at once, replaces nothing. A space upgrade is stamped with the target space's uuid (`store.WithSpaceUUID`) so it restores over that space. Only drives `store`, so HTTP may import it, never `legacy` itself. `backup` and `ledger` know nothing of Laravel.

## HTTP

- Domain types free of JSON/HTTP. `httpserver/dto` converts: camelCase keys, optional values present as `null`, money as JSON number at stored scale.
- New route: update `httpserver/apidocs/openapi.yaml`. Test fails if route and spec differ.
