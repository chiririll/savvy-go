# Plan: slim transaction list with a total amount

Status: idea, discussed 8 Nov 2026, decisions below agreed with the owner. Nothing is decided in code.

Checked in code (8 Nov 2026): `currencies.rate` is base-currency units per one unit of the currency (`ledger.Currency.ConvertToBase` is `amount * Rate`); conversion between two currencies already exists as `ledger.Convert` / `ledger.TryConvert` (`internal/ledger/currency.go`) and must be reused. **Payments have no order**: table `payments` has primary key `(transaction_id, account_id)` and no position column (`internal/migrate/sql/space/0001_base.sql`), and `ListPaymentsOfTransactions` sorts by `account_id`. So "first payment" is not defined yet, see the open questions.

## Goal

The list endpoint returns a transaction that is ready to draw: no sums, no conversion in the frontend. Everything else (all payments, lines with notes, `restrictions`, `difference`) is loaded when the transaction is opened. The MCP server will later be reworked to ask for the fields it needs.

This is a UI optimization. It is not a new accounting figure.

## Today

- `GET /transactions` and the single transaction return the same `Transaction` (`resources/ts/types/transactions.ts`), with `payments`, `lines`, `tags`, `actions` and `restrictions`.
- `amount` is what the payments net to, and `null` when they span several currencies. The list cannot show a total for those.
- The form sums payments in the browser (`paymentTotals` in `resources/ts/lib/transaction-payments.ts`) with JS numbers rounded per currency. That stays for live editing, but the list should not need it.

## Design

### Three fields, filled for every transaction

| Field | Meaning |
|---|---|
| `total_amount` | Net of change, in `total_currency`. Never null while a rate exists: for a single-currency transaction it is the exact amount, for a multi-currency one it is converted with the current rates. |
| `total_currency` | Code of the currency of `total_amount`: the one the payments share, or, when they span several, the currency of the **first payment**. |
| `total_flags` | List of strings. `multi_currency`: the payments span several currencies. `approximate_rate`: `total_amount` was converted with the current rates, so it is an estimate. |

Names are provisional. The space's base currency does not decide the target currency: a transaction is converted against its own first payment (rates are stored against the base, so the conversion goes through it, `from / to`). The UI draws "≈" when `approximate_rate` is set. The original amounts stay in `payments`, which the detail view loads.

The existing `amount` field keeps its meaning (`null` for several currencies) for existing clients. See "Slim list and detail".

### Stored on the transaction, written when payments are saved

- **The point is to avoid loading every payment of every listed transaction.** So the total is stored on the transaction row and the list is a plain read. Precision is not a goal: the value is a display hint.
- New columns on `transactions`: the total in **integer minor units** (no float), the currency of that total (a currency id), and a `multi_currency` flag. `flags` in the API is built from them (`approximate_rate` is set when `multi_currency` is).
- Computed with the **current** rates (`currencies.rate`) at the moment the payments are saved. No historical rates, no rates table, no provider setting, no network call.
- **Rate changes do not recompute anything.** A stored estimate is as old as the last save of that transaction. That is accepted.
- Convert each payment into the target currency, round to its decimals, then sum (one rule, written down), in Go with `decimal`, through the existing `ledger.TryConvert`. Do not copy the SQL `CAST(c.rate AS REAL)` arithmetic from `internal/db/filter/filter.go`: it is float.
- If a currency has no stored rate, the stored amount is `NULL` and the UI shows "no estimate". A single-currency transaction never needs a rate.

### When it is recomputed

- Payments edited, including change rows and the first payment changing.
- A recurring occurrence is confirmed (the real payments are set).
- Every other writer of `payments` (for example a transfer mirrored from another space): check that all of them go through the one place that recomputes.
- Not on rate updates and not on base currency change.

### Display only

Validation (the net-sign check), reports and budgets keep using exact conversion. **Nothing may read the stored total except the list.** Say so where the field is built and in `docs/`.

### Out of scope

- **Backfill of rates and historical rates.** Not needed.
- **Existing rows.** Edit `0001_base.sql` in place (agreed). A database that already ran 0001 does not pick the columns up, so decide how existing rows are filled: see the open questions.
- **Import.** It does not support multi-currency yet. Revisit this plan when it does.

## Slim list and detail

- List item: id, type, date, status, description, `total_amount`, `total_currency`, `total_flags`, main categories, account names, tags, and the flags the row needs (`actions`, needs attention).
- Detail (`GET /transactions/:id`): the full object, loaded when the dialog opens. Prefetch on hover. The edit form reuses it.
- Open question: things that act straight from the list (duplicate, confirm, "transfers needing attention") may need the detail first or an endpoint of their own.
- Prefer **one serializer with a whitelist of fields** (`fields=` parameter, default slim on the list) over two separate shapes, so list, detail and MCP cannot drift apart. The MCP rework uses the same parameter.
- Do not change the current shape for existing clients: add the parameter, keep the old default until the frontend moves.

## Tests (write first)

- Single-currency transaction: `total_amount` is the exact amount, `total_currency` is its currency, no flags.
- Multi-currency transaction: `total_currency` is the first payment's, `total_amount` converted with the current rates, flags `multi_currency` and `approximate_rate`.
- Change nets out of the total.
- Rounding: each payment is rounded to the target currency's decimals, then summed (a case where rounding the sum would differ).
- Changing a rate changes the estimate; changing the space's base currency does not change the target currency.
- Multi-currency with a currency that has no rate: `total_amount` null, no error. Single-currency never needs a rate.
- Editing payments, and confirming a recurring occurrence, rewrites the stored total.
- A rate update does not change stored totals.
- Every code path that writes `payments` ends with a consistent stored total.

## Docs

Update `docs/transactions.md` and `docs/api.md` (new fields, flags, estimate meaning, current rates), and `docs/mcp.md` when the field selection lands.

## Open questions

- **Which payment is "first"?** Payments are stored unordered (see Status), so the order the user typed in the form is lost on save. Options: (a) add a `position` column to `payments` (edit 0001 in place) and have the form send the order; (b) a rule that needs no schema, for example the lowest `account_id`, which is stable but arbitrary for the user; (c) the currency of the largest payment by absolute value in the base currency. Decide before building.
- Change rows are payments of the opposite sign: the target currency must come from a payment of the transaction's own sign, not from a change row.
- Should `approximate_rate` be its own flag? With current rates only, it is true exactly when `multi_currency` is true. It earns its place only if the two can differ (for example later, when a dated rate exists).
- **Existing rows and 0001.** Editing 0001 in place leaves already-created databases without the columns, and single-currency rows now also need a total. Fill them with a small startup step, or accept that dev databases are recreated.
- Should the list sort or filter by `total_amount`? A stored column could be indexed, but default is still no.
- Does the frontend still need `paymentTotals` outside the form?
