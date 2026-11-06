# v1.1 schema: everything except the transaction rework

Status: notes from 29 Oct 2026, nothing decided. The transaction/payment rework lives in [payments-split.md](payments-split.md) and is not repeated here. Items come from [backlog.md](backlog.md).

Rule of thumb: a new nullable column or a new table is cheap and can also go into a later migration (0003). Only table rebuilds must be done before release, and the only one is the transaction rework. So nothing here blocks it.

## Likely for v1.1

### Subcategories (#84)

- `categories.parent_id INTEGER REFERENCES categories(id) ON DELETE SET NULL`, one level only (the depth limit is checked in `internal/ledger`, not in SQL).
- Replace `categories_type_name_unique` with a unique index on `(type, COALESCE(parent_id, 0), name COLLATE NOCASE)`, so two parents can each have "Other".
- A budget on a parent covers its children in the query. `budget_category` does not change.
- Sync groups copy categories between spaces: the parent link must sync too.
- Reports roll children up into the parent and allow drill-down (#83).

### Instance invitations (#107)

- Lives in the **server** database, so a new `server/0002`, not a space migration.
- Option A from the backlog: table `instance_invitations` (token hash, role, expiry, use limit, optional note, revoked_at). No change to `users`.

### Adjustments (#88, #89)

- Own table, not a transaction type. Shown as a grey note on the day header, with a page to see all of them. Not in the transaction list, not in reports.
- Sketch: `adjustments (id, account_id, date, amount, note)`. It counts in the account balance, so balance = payments + adjustments + `initial_balance`.
- Reconciliation points ("04.09, actual 5 586") are probably the same table plus an `actual_balance` column. Design later.

## Open, needs a decision first

### Report currency and rate history

- Today only `currencies.rate` (the current rate) exists.
- Converting a report "at the rate of the transaction date" needs a table `currency_rates (currency_id, date, rate)`. History cannot be rebuilt afterwards, only started from the day it exists.
- The transaction rework reduces the problem: category amounts are exact in the bill currency. Only base-currency totals and balance history still convert.
- Question: today's rate or historical? If historical, create the table in 0002 and seed one row per currency.

## Not now (needs design, additive later)

- Who paid and for whom, monthly "who owes whom" (#105): probably a column on payments. Users live in the server DB, with no foreign key.
- Events and trips (#93): tags already cover most of it. Alternative: a parent link on transactions.
- Exchange linked to the purchase it paid for (#95): same options as events.
- Two-way budgets (#85) and budget instalments (#86): depend on the currency question above.
- MCP rollback (#102): probably a `batch_id` on transactions. The unit of rollback is undecided.
- Count by denominations (#90).

## No schema change

Filter by currency (#71), balance at a date and balance history (#87, an index question only, see Performance in the split plan), compact `list_transactions` (#101), batch create (#100), import by receipt link (#98), quick entry (#97).
