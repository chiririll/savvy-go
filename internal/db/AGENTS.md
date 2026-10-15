# Database layer

```bash
go generate ./internal/db   # sqlc: queries -> internal/db/sqlc (generated, never edit)
```

- SQL only in `internal/db` (+ `sqlc`, `filter`), `internal/db/repo/<area>`, `store/sqlite`, `migrate`, `legacy`. Business code never builds query or sees `sql.Null*`.
- Repository speaks plain Go: `time.Time`, pointers for optionals, `money.Money`, nil for no row, bool for "changed a row". Decides nothing. Rules and clock (handed `now`) stay in business code.
- Package imports only own repository. Other area's rows come through that area's package. Exceptions: `reports`, `seed`.
- New query: add `.sql`, generate, add repo method, call it. Test method with `db/dbtest`.
- Server queries touch only server tables, space queries only space tables. Keep SQL Postgres-compatible.
- sqlc sqlite: use each `narg` once per query (COALESCE/CASE), never `OR col = ?N`. Driver counts every `?`.
- Dynamic SQL (IN lists, report GROUP BY) only in `internal/db/filter`.
