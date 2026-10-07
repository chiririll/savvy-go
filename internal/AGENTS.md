# Backend (Go)

```bash
go generate ./internal/db   # sqlc: queries -> internal/db/sqlc
go run ./cmd/savvy-go       # serves :8080, data in ./data
```

## Store boundary

- `domain`, `httpserver`, `auth`, `settings`, `seed`, `signing`, `schedule`, `jobs` use storage only via `internal/store`. No `store/sqlite`, `migrate`, `legacy`, `modernc.org/sqlite` imports. No `PRAGMA`, `ATTACH`, `sqlite_master` in SQL. `store/arch_test.go` enforces. Reason: space DB may become Postgres schema later.
- Work touching two spaces: `store.InSpaces`.

## SQL

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

## HTTP

- Domain types free of JSON/HTTP. `httpserver/dto` converts: camelCase keys, optional values present as `null`, money as JSON number at stored scale.
- New route: update `httpserver/apidocs/openapi.yaml`. Test fails if route and spec differ.
