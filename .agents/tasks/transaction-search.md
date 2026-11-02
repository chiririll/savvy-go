# Transaction search: remaining work

Issues: [#72 free-text search](https://github.com/chiririll/savvy-go/issues/72),
[#70 filter by account](https://github.com/chiririll/savvy-go/issues/70).
Nothing is committed. Backend is written; the frontend and one rename remain.

## Done (backend)

`GET /api/transactions` and the MCP `list_transactions` tool:

- `q`: free text. Space-separated terms, all must match. A term matches the
  description or an item name (case-insensitive, any script), and a number also
  matches `amount`, a transfer's `to_amount` or an item price, **in the
  currency of its account** (stored minor units depend on the currency's
  decimals). "1 879", "1879", "18.79", "18,79", "1,879" are read as numbers
  ("1,879" as 1879 and as 1.879). Capped at 200 runes and 8 terms.
- `account_ids=1,2`: source account is any of these (same rule as `account_id`).
- `currency=usd,jpy`: codes, any case. A transfer matches if either account's
  currency is listed.
- `with_summary=1`: now totals the transactions matching the same filters
  (confirmed income and expense, base currency). The standalone summary
  endpoints are unchanged and unfiltered.

Code: `internal/db/filter/search.go` (parser and clause), `filter.go`
(`TxFilter`, `txWhere`, `SummaryRows`), `internal/db/funcs.go` (SQL functions
`savvy_fold`, `savvy_pow10`), handler `internal/httpserver/api/transactions`.

Not yet confirmed: `TestListSummaryFollowsTheSearch` in
`internal/httpserver/transactions_search_test.go` (build tag `integration`).
Run it: `go test -tags integration -run TestListSummaryFollowsTheSearch ./internal/httpserver/`.

## Task 1: tidy the tests

`internal/db/repo/ledger/search_test.go` also tests the `currency` and
`account_ids` filters, and the name does not fit that package's naming. Rename it
to `transactions_filter_test.go` (`git mv`). Nothing else changes. The parser's
own pure tests stay in `internal/db/filter/search_test.go`.

## Task 2: frontend

Decisions already made with the user:

- **Keywords are parsed on the client** into the structured API params. The API
  stays id-based; `q` stays plain text.
- Keywords: `account`, `currency`, `category`, `tags` (plural), `type`.
  Comma-separated values: `tags:food,coffee`. Quote values with spaces:
  `account:"Cash USD"`. Repeating a key adds values. Within a key the values
  are OR (`tag_ids` and `category_ids` already behave this way); different
  keys are AND.
- A name that resolves becomes a chip. An unknown key or an unresolved name
  stays in `q` as plain text.
- Keywords, the filter panel and the palette share **one** `TransactionFilters`
  state: what the palette sets must show in the filter panel (user: "current
  filters should be written in panel").
- **Palette** opens with **`/`** (not Ctrl+K: Firefox owns it; not Ctrl+P). Only
  when no input/textarea/select/contenteditable is focused, no modal is open,
  no Ctrl/Meta/Alt, not a key repeat, not during IME composition. Call
  `preventDefault` (Firefox Quick Find). Match on `event.key === "/"`.
  Desktop only: no palette on mobile.
- Desktop header gets a search button with a `/` hint that opens the palette.
- Palette: autocomplete keywords, then values from the lists the SPA already
  holds (accounts, categories, tags, currencies). Tab accepts. Live preview from
  the list endpoint, small `per_page` (~8), debounced, abort stale requests,
  show `with_summary` totals (base currency). Enter opens the selected result;
  Ctrl+Enter (or a "show all N" row) applies the filters to the list page via
  URL params. Esc closes. Combobox/listbox ARIA, focus trap.
- Mobile: a plain search field in the filter panel that sends `q` (open
  question: whether the keyword parser also runs there; it is a pure function,
  so it is cheap to allow).

Steps, each test-first:

1. Keyword parser as a pure function with unit tests (`npm test`).
2. Extend the types, `api/transactions.ts` and `hooks/use-transaction-list-filters.ts`
   with `q`, `account_ids`, `currency`.
3. Account control in the filter panel (#70; the issue names
   `TransactionFilterDetails.tsx`, locate it under `components/features/transactions/list/`),
   plus the search field.
4. The palette and the `/` shortcut (the "should open" check as a pure function
   with unit tests).
5. Update `docs/` (user-visible), including the keywords and the `/` shortcut.

Not verified when written: exact component paths above, and how the list page
shows the summary totals.

## Other loose ends

- The MCP `list_transactions` tool has `q` but not `account_ids` or `currency`.
- `go generate ./internal/db` pins sqlc v1.30.0 but the repo was generated with
  v1.31.1; it rewrites every generated file's version header. Use the local
  sqlc (`cd internal/db && sqlc generate -f ../../sqlc.yaml`) or align the pin.
- Stories and browser tests run on the remote only. Go tests, `tsc` and unit
  `vitest` (`npm test`) run locally.
