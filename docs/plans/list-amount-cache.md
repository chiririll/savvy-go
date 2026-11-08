# Plan: slim transaction list with an approximate amount

Status: idea, discussed 8 Nov 2026. Nothing is decided in code. Rate storage in `internal/currencydata`, `internal/refsync` and `internal/money` was **not checked**: read it before fixing the schema.

## Goal

The list endpoint returns a transaction that is ready to draw: no sums, no conversion in the frontend. Everything else (all payments, lines with notes, `restrictions`, `difference`) is loaded when the transaction is opened. The MCP server will later be reworked to ask for the fields it needs.

## Today

- `GET /transactions` and the single transaction return the same `Transaction` (`resources/ts/types/transactions.ts`), with `payments`, `lines`, `tags`, `actions` and `restrictions`.
- `amount` is what the payments net to, and `null` when they span several currencies. The list cannot show a total for those.
- The form sums payments in the browser (`paymentTotals` in `resources/ts/lib/transaction-payments.ts`) with JS numbers rounded per currency. That stays for live editing, but the list should not need it.

## Design

### Two amounts

| Field | Meaning |
|---|---|
| `amount` | Exact. Net of change, in the one currency the payments share. `null` when they span several currencies (as today). |
| `approximate_amount` | Net of change, converted to the base currency of the space. Filled for every transaction, single-currency ones included, so the list can sort and compare without branching. For display only. |

The original amounts stay in `payments`, which the detail view loads.

The API says which currency `approximate_amount` is in (the space's base currency, not the viewer's). The UI marks it as an estimate (for example "≈") when `amount` is `null`.

### Column `amount_cache`

- New column on the transaction table: the approximate amount in **integer minor units** of the base currency. No float column. Precision matters little for display, but minor units cost nothing and follow the money rule.
- Computed with the rate **on the transaction date**, not the current one, so past rows do not change from day to day and the list is a plain read from the DB.
- Rate source: use the rates the app already stores, if there are any. If there are none, fetch them in a **background job**, never in the request path, because the app is self-hosted and may be offline. The historical rates API is:
  `https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@2024-03-06/v1/currencies/eur.json` (base currency in the path, date in the version).
- Convert each payment, round to the base currency's decimals, then sum (one rule, written down, so list and reports agree where they should).

### When the cache is recomputed

- Payments or date edited.
- A recurring occurrence is confirmed (the real date and payments are set).
- The base currency of the space changes: every row.
- Backfill once for existing rows (migration or job).

### When there is no rate

The CDN may lack a currency or a date. Leave `amount_cache` null, fall back to the current rate if there is one, else send no approximation. The UI needs a "no estimate" state for it.

### Display only

Validation (the net-sign check), reports and budgets keep using exact conversion. **Nothing may read `amount_cache` except the list.** Say so in the column comment and in `docs/`.

## Slim list and detail

- List item: id, type, date, status, description, `amount`, `approximate_amount`, main categories, account names, tags, and the flags the row needs (`actions`, needs attention).
- Detail (`GET /transactions/:id`): the full object, loaded when the dialog opens. Prefetch on hover. The edit form reuses it.
- Open question: things that act straight from the list (duplicate, confirm, "transfers needing attention") may need the detail first or an endpoint of their own.
- Prefer **one serializer with a whitelist of fields** (`fields=` parameter, default slim on the list) over two separate shapes, so list, detail and MCP cannot drift apart. The MCP rework uses the same parameter.
- Do not change the current shape for existing clients: add the parameter, keep the old default until the frontend moves.

## Tests (write first)

- Single-currency total: `amount` set, `approximate_amount` equal to it.
- Multi-currency total: `amount` null, `approximate_amount` converted with the rate of the date.
- Change nets out of both amounts.
- Edit of payments or date recomputes the cache.
- Base currency change recomputes every row.
- Missing rate leaves the cache null, no error.
- A past row does not change when the current rate changes.

## Docs

Update `docs/transactions.md` and `docs/api.md` (new fields, estimate meaning), and `docs/mcp.md` when the field selection lands.

## Open questions

- Do rates already exist in the DB, and in which table?
- Which currencies does the CDN not cover (crypto, rare codes)?
- Should the list sort or filter by `approximate_amount`?
- Does the frontend still need `paymentTotals` outside the form?
