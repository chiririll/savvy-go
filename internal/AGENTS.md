# Backend (Go)

```bash
go generate ./internal/db   # sqlc: queries -> internal/db/sqlc
go run ./cmd/savvy-go       # serves :8080, data in ./savvy-data
```

## Store boundary

- `ledger`, `spaces`, `transfers`, `uploads`, `automation`, `recurring`, `reports`, `calendar`, `currencydata`, `sso`, `oauth`, `backup`, `importer`, `legacy/restore`, `httpserver`, `auth`, `settings`, `seed`, `signing`, `schedule`, `jobs` use storage only via `internal/store`. No `store/sqlite`, `migrate`, `legacy`, `modernc.org/sqlite`, `database/sql`, `db/sqlc` imports. No `PRAGMA`, `ATTACH`, `sqlite_master` in SQL. `store/arch_test.go` enforces. Reason: space DB may become Postgres schema later.
- Work touching two spaces: `store.InSpaces`.

## Packages

Dependencies point down. Each package may import the ones above it in this list, never below.

- `calendar`, `money`, `signing`, `db`, `store`: leaves.
- `mailer`: leaf. Sends plain-text email over SMTP (`starttls` never downgrades to plain). Configured by `[mailer]` in config.toml (`config.Config.Mail` at start), never the DB. The admin panel edits it: `PUT /api/admin/mail` validates (`mailer.InvalidError` names the field), writes the file (`config.Live.SetMail`) and swaps the live mailer, no restart. `httpserver` keeps the current settings and mailer under `mailMu`: read them through `s.mailer()` / `s.mailSettings()`, never `cfg.Mail`. A missing `password` keeps the stored one. A bad section at start disables mail (`Config.MailProblem`), it does not stop startup. `WithMailer` in tests; a test that saves swaps `s.newMailer`. Email is a convenience: invitations always return the link, a failed send never undoes them.
- `currencydata`: the one place that talks to the currency data service (names, rates). One `Source` per server, shared by all spaces. Update rates are cached in a map by base currency (spaces with the same base share a request; the job calls `Expire` once per run); catalog look-ups fetch live and skip that cache. `ledger` only sees the `ledger.RateSource` interface. Tests use `currencydata.NewOffline()`.
- `bootstrap`: leaf (stdlib only). First-run password that `POST /api/auth/register` needs while there are no users. `Guard.Register` verifies, creates and invalidates under one mutex, so parallel registrations make one admin; `Verify` checks without spending it (`POST /api/auth/bootstrap/verify`, so the setup page can refuse a wrong one on screen 1); a nil guard is closed. `cmd/savvy-go` creates it only when the user count is 0 and passes it via `httpserver.WithBootstrap`; tests use `testApp.bootstrap`. First-run restore (`POST /api/onboarding/restore`, `httpserver/onboarding_restore.go`) runs inside `Register`, so a failed restore keeps the password and a backup with no users is refused (the server would have nobody to log in); the uploaded backup is deleted on failure. `Invalidate` is for any other route that creates the first user. Auth routes are deliberately not in `openapi.yaml`.
- `spaces`: registry, members, roles, invitations, quotas, audit log. Needs from outside go through small interfaces (`spaces.FinalBackuper`).
- `ledger`: finance core of one space: accounts, transactions, categories, tags, currencies, budgets, debts. Knows nothing of the packages below.
- On top of `ledger`, independent of each other: `reports`, `recurring`, `automation`, `importer` (CSV; also uses `uploads`), `uploads`.
- `transfers`: signed transfers between linked spaces. Uses `spaces`, `signing`; not `ledger`.
- `backup`: backup files, archive format, restore. Uses `spaces`, `transfers`, `signing`.
- `sso`: identity providers, OIDC/OAuth/SAML, JIT provisioning. A new user gets no space.
- `oauth`: authorization server for MCP clients (Claude web). Dynamic registration of public clients, closed unless an admin opened a 5-minute window (in memory, closes after one registration; closed, `/oauth/register` is a 404 and the metadata omits `registration_endpoint`, so nothing hints it exists). Capped, idle clients pruned, PKCE `S256` only, rotating refresh tokens. A client stores who registered it (`ip`, `user_agent`, `origin`, also logged). There is no per-app approval: the admin window plus each user's consent are the gates. The admin always sees the list and can remove a client (`/api/admin/oauth/clients`, session-only): its access tokens are deleted with it. The consent screen makes the user pick the spaces (or all); an access token is a `svy_` API token with a 1h expiry, grants on the `auth.MCPResources` the user ticked (consent sends `permissions`, validated by `mcpPermissions`: any level, only those resources; default read on all, write/delete never by default; carried through refresh). The consent page confirms again before issuing any write or delete grant and those spaces, which every refresh carries over, so it shows in the user's token list and respects roles; the refresh row cascades from `api_tokens`, so revoking there ends the connection. Auth codes live in memory (60s, single use, process-local). The consent API (`/api/oauth/authorize`) is session-only: a token must never mint tokens. Needs `app_url`: it is the issuer.
- MCP server (`httpserver/mcp.go`, `/mcp`, stateless): tools with no data access of their own. Each one calls the app's own route in-process (`mcpCall`) with the caller's bearer token, so roles, token spaces and permissions and the DTO/money shape stay those of the REST API. A new tool's route must fall under a resource in `auth.MCPResources`, or connected assistants get a 403 (`TestMCPToolsNeedTheirPermissions`). Read tools (`list_*`/`get_*`, `mcp.go`) validate their inputs into a GET query. Write tools (`mcp_write.go`) are one typed arg struct plus a `writeTool` line each: the args become the JSON body, `{space_id}`/`{id}` fill the path, so the route's own validation and the token's `write`/`delete` level decide; a read-only token gets the REST 403 naming the missing permission. Money args are strings, never JSON numbers. Delete tools carry `DestructiveHint`, every write tool leaves `ReadOnlyHint` off (`TestMCPToolsAreMarkedByWhatTheyDo` keys on the `list_`/`get_` prefix). Only resources in `MCPResources` get write tools (no recurring, automation, transfers, space settings). The inner call needs a fresh chi route context, or it inherits the POST of `/mcp`.
- API tokens (`auth/api_token.go`, `permissions.go`): least privilege, fail closed. A token holds `<resource>:read|write|delete` grants (each level includes the ones before; delete is separate so a token can add without removing; the HTTP method decides the level, `auth.LevelFor`; resources in `auth.APIResources`), or a preset (`auth.PresetPermissions`: viewer, editor, admin) that the create endpoint expands and either a list of spaces (`api_token_spaces`) or `all_spaces`. Creating one needs at least one space, a space outside the list is a 404 like a non-member's (`withSpace`), `GET /spaces` lists only the token's. `requireTokenGrant` maps the first path segment after `/spaces/{id}` to a resource through `spaceSegments`; a segment not in the map is closed to tokens, and `TestEveryDataRouteMapsToAResource` fails when a data route has no entry. Grants never exceed the owner's role, checked per request, never at creation. Changing categories or tags of a space in a sync group (they mirror into the others) needs a token that reaches every space of the group (`tokenReachesSyncGroup`), and `refsync` still checks the owner's role in each; sync groups themselves are `sessionOnly`. Space administration, backups and credentials stay `sessionOnly`. A token with no grants can only list its spaces.
- Create-space request: needs `spaces` and `ledger` together, so it lives in `httpserver/spaces_fill.go` (`createSpace`). All or nothing. No code may assume a user has a space.
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
