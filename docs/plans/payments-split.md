# Plan: transactions with payments and lines

Status: design agreed in conversation on 29 Oct 2026, not implemented. Target: v1.1, as a rewrite of `internal/migrate/sql/space/0002_transactions.sql`. The schema is in [transactions-erd.md](transactions-erd.md); this file covers the reasons, the rules, the migration and the work. Other schema ideas are in [schema-v1.1-other.md](schema-v1.1-other.md).

0002 may be rewritten because it is unreleased: `v1.0.0` and `v1.0.1` do not contain it. Its current content (`day_order`, `group_uuid`) is deleted; `group_uuid` never shipped, so there are no groups to convert. `0001_initial.sql` is frozen, because v1.0 databases exist.

## Why

Today one row of `transactions` is one account, one currency and one category, with a second account and amount (`to_account_id`, `to_amount`) for transfers. A purchase paid from several accounts or currencies, or with several categories, does not fit. Split payments are a workaround: several ordinary rows that share a `group_uuid`, each repeating the description and category, with the change stored as an income row so it counts as income in reports.

Goal: a transaction is one event (date, description, tags) with **payments** (the money that moved, per account) and **lines** (what it was for, per category and currency). Multi-currency is then the normal case. Go Savvy is a budget tracker, not a receipt store: a line is an amount for a category, not a product.

## Decisions

| Topic | Decision |
|---|---|
| Split | `transactions` (the event), `payments` (money moved), `transaction_lines` (what it was for). Groups and `group_uuid` are removed. |
| Facts only | Every amount is an exact integer in minor units of its own currency. No rate is stored. Payments and lines are not tied by a database constraint. |
| Payments | Signed, in the account's currency, one per account in a transaction. Change is a payment with the sign opposite to the transaction's direction, on another account. No type column: the role follows from the sign and the transaction type. |
| Lines | Category, currency (one of the currencies of all payment accounts, change included), amount, optional note. Several lines may share a category. "Bus 100" is one line; a receipt is one line per category, or one per product with the name in the note. |
| Items | Removed: no catalogue, quantity or unit price. v1.0 item names are kept in the line's note. |
| Transfers | Two or more payments, and optional lines with expense categories for a fee. |
| Keys | `payments (transaction_id, account_id)` and `transaction_lines (transaction_id, seq)`, both `WITHOUT ROWID`: clustered by transaction, no `id`, one index fewer each. |
| Estimates | Only on recurring rules: `"estimated": true` on a payment in the template. A pending occurrence shows it through its rule. Transactions have no flag. |
| Status | Stays on `transactions`; every report filters on it. |
| Recurring, what | One `template` JSON: the transaction input of the API (type, description, payments, lines, tags), so a rule can create any transaction that can be entered by hand. No mirror tables; `recurring_transaction_tag` is dropped. The ledger does what foreign keys would. |
| Recurring, when | One `schedule` JSON (frequency, interval, days, start, end). `next_run_date` and `last_run_date` are dropped and derived from `recurring_instances`. There is no scheduler: a rule always has one pending occurrence, and confirming or skipping it creates the next. |
| External id | `transactions.external_id`, nullable, not unique, importer adds a prefix. Replaces `dedup_hash`. A partial index serves the lookup. |
| Locks | `locks` (one row per managed transaction, with a `kind`) and an owner table per kind (`space_transfer_links`, `debt_links`). Replaces `guardRow`, `ErrGroupLeg` and `space_transfer_uuid`. |
| Debt rows | Migrated with one payment and a `debt` lock; `debt_links` keeps the debt account and `to_amount`, so nothing is lost while debts are parked. |
| Zero amounts | No line of 0 (`CHECK (amount > 0)`). 0 payments from v1.0 are kept, because they hold the account; the ledger rejects 0 on new input. |
| Deletes | Accounts and categories used by payments or lines are restricted. |
| Order in a day | `day_order` stays unchanged. |

## Semantics

Validation lives in `internal/ledger`, not in the database, because currencies differ.

| Transaction type | Payments | Lines |
|---|---|---|
| `expense` | negative on the accounts that paid; positive for change | one or more, positive, expense categories |
| `income` | positive; negative for change | one or more, positive, income categories |
| `transfer` | at least one negative and one positive, each in its own account's currency | none, or fees with expense categories |
| `transfer_out`, `transfer_in` | one payment, written only by the cross-space code | none |

- **One account once.** A transaction has at most one payment per account (the primary key). The form merges or refuses a second row for the same account.
- **Net sign.** The ledger refuses an expense whose payments' net is positive, or an income whose net is negative: change larger than the payment is always a mistake.
- **Balance** of an account: `initial_balance + SUM(payments.amount)` over its payments in `confirmed` transactions. No `CASE` on type. This replaces `AccountDailyDeltas`.
- **Spending, income reports and budgets** read the lines, joined to the transaction for date and status. They never read payments, so change is never income and a transfer fee is spending. Base-currency totals convert each line by its currency's rate.
- **Soft check** when saving: convert the payments' net and the lines' total to one currency at the current rates (each currency's total once, rounded once, half-even) and warn if they differ by more than a tolerance. It never blocks. Stored data is never converted. It runs only when payments or lines changed in this save, so an old cross-currency transaction does not warn just because rates moved.
- **Form.** One account and one line is the simple form; amounts stay in sync while they share a currency. Removing an account whose currency some lines use is refused until those lines change currency.
- **Automation rules** run once per transaction. Their JSON keeps its shape; the meaning of fields changes: `amount` is the payments' net, compared within one currency; `account_id` matches any payment; `category_id` matches any line; `set_category` sets uncategorised lines only.
- **Import**: a bank line is one transaction with one payment and one line. The importer sets `external_id` and, for a source without a natural id, finds candidates by account, amount and date for a "possible duplicate" prompt.
- **Locks**: see the ERD. `Transaction.Allowed()` is the only place that decides what may be done. The owner deletes its transactions: `internal/transfers` deletes the mirrored transaction before a `space_transfers` row, because the link no longer cascades from it.
- **Lines**: the ledger checks that a line's category type matches the transaction type and that its currency is one of the payment accounts' currencies.
- **Recurring**: one pending occurrence per rule holds by construction, and `UNIQUE (rule, due_date)` stops crash recovery from creating the same occurrence twice. The next due date is one step of `schedule` after the latest `due_date`. Generating an occurrence validates the template and creates it like a transaction entered by hand. At most one payment may have a NULL amount: it balances the rest at that day's rates and is estimated (a transfer's receiving side, or a subscription priced in USD and paid from an RSD card). An account used by a template cannot be deleted; deleting a category or a tag removes it from templates; a template that still fails stops its rule (`is_active = 0`, reason in `error`). The startup job `recurring:ensure-upcoming` validates the template and schedule of every active rule before creating anything. Deleting a rule keeps its transactions; a pending one becomes an ordinary pending transaction. Deleting an occurrence keeps its instance row (`transaction_id` set to NULL), so the next date never moves back. - **Duplicates**: `external_id` is not unique, so only the importer prevents a double import (one process, so no race).

## Migration 0002

Rewritten from scratch. It converts rows in the 0001 shape. One script, run by `migrate.execScript` in a single transaction with `foreign_keys=ON`.

**Do not `DROP TABLE transactions` with children in place.** `transaction_items` and `transaction_tag` reference it with `ON DELETE CASCADE`, and `PRAGMA foreign_keys` cannot be changed inside a transaction, so the drop would delete them. Steps:

1. Remember `sqlite_sequence.seq` of `transactions` and `recurring_transactions` (ids must not be reused).
2. Copy to staging: `transactions`, `transaction_items`, `transaction_tag`, `recurring_transactions`, `recurring_transaction_tag`.
3. Drop the five originals (their indexes go with them), create the new tables.
4. Insert `transactions` with the same ids. Insert payments and lines by the table in the ERD. Each income or expense with an amount above 0 gets one line (category from the transaction, account currency, the amount); its note is the trimmed item names in item id order, joined with `, `, or NULL. A row with amount 0 gets a payment of 0 and no line. A transfer from an account to itself gets one payment of the net. `is_estimated` is dropped. `dedup_hash` becomes `external_id` with the `csv:<account_id>:` prefix.
5. Re-insert tags. Fill `recurring_instances` from `recurring_transaction_id` (`due_date` = `substr(date, 1, 10)`; `INSERT OR IGNORE`, so a second transaction of one rule on one day has no link). Insert each rule with the same id, its `schedule` from `json_object` of the old schedule columns, and its `template` from `json_object` like a transaction of its type (`is_estimated` on the payments; a NULL `to_amount` stays NULL; tags from `recurring_transaction_tag` with `json_group_array`). The pending occurrence's `due_date` is the rule's old `next_run_date` (its transaction date may have been edited); a rule with no instance gets `start` = `next_run_date` when that is later. Then the derived next date equals the old one.
6. Create `locks` for every `transfer_in`, `transfer_out` and `debt_*` row; fill `space_transfer_links` from `space_transfer_uuid` and `debt_links` from `to_account_id` and `COALESCE(to_amount, amount)`. The `category_id` of debt rows is dropped (the app never sets it).
7. Restore the sequences, drop staging, create the indexes.

## Laravel imports stop at 0001

A Laravel-era file is brought to the **0001 shape and no further** by `internal/legacy`. After that it is an ordinary database, and the normal migrator applies 0002, 0003 and so on in sequence. 0002 knows nothing about Laravel.

The code does not do this yet. Today `legacy.Upgrade` (`internal/legacy/legacy.go`) calls `upBoth`, which applies **all** migrations to the Laravel-shaped tables *before* `convert` turns decimal money into minor units and dates into text. `conform` then rebuilds the tables to the shape of all migrations (`targetTables` in `conform.go`), and `EnsureIndexes` replays every `CREATE INDEX` of every file. A rewritten 0002 would run on unconverted data, and 0001's indexes on columns that 0002 drops would fail.

Work item, done first:
- `migrate` gets a way to apply only the baseline (`0001`) of a set, and to list only the baseline's indexes.
- `legacy.upBoth`, `targetTables` and `ensureIndexes` use the baseline only.
- After `Upgrade`, `schema_migrations` records only `0001`, so the startup migration runs 0002 on the converted data.
- Test: a Laravel fixture goes through `Upgrade` and then `Up`; balances and report totals match the old file. The existing legacy tests pass unchanged.
- `extraColumns` keeps producing exactly what 0001 expects.

## Performance

A spending report joins `transaction_lines` to `transactions` (date, status); a balance joins `payments` to `transactions`. Primary-key lookups are cheap, but do not guess:

1. Build the schema normalised, with the indexes in the ERD.
2. Seed a large space (about 200k transactions) with the demo seeder. Compare balances, the report queries, the budget queries and the transaction list on the old and the new schema.
3. Only if the numbers are worse, copy `date` and a `confirmed` flag onto `transaction_lines` and `payments` and index them. Keep them in sync in the repository code, not with triggers: `internal/db/AGENTS.md` asks for Postgres-compatible SQL. A category cache on `transactions` is not an option: a transaction can have several categories.

The transaction list pages the headers by date, then loads payments, lines and locks by `IN (ids)` in one query each.

## Code affected

- `internal/db/queries/space/*.sql` and `internal/db/repo/*`: `transactions.sql`, `accounts.sql` (balances), `transfers.sql`, `recurring.sql`, `imports.sql`, `budgets.sql`, `categories.sql` (counts and reassign on lines); regenerate with `go generate ./internal/db`. The sqlc rule: use each `narg` once per query.
- `internal/db/filter/filter.go`: transaction lists and report grouping.
- `internal/ledger`: `transaction.go` (`Allowed()`, delete `guardRow`), `account.go`, `budget.go`, `hydrate.go`, `related.go`; delete `group.go` and the item code.
- `internal/automation`: field meanings in `conditions.go`, `set_category` in `automation.go`.
- `internal/recurring`, `internal/importer`, `internal/transfers`, `internal/reports`, `internal/seed`.
- `internal/httpserver`: DTOs (`dto/transactions.go`, item fields go), routes, MCP tools in `mcpserver/write_tools.go` (replace the three `*_transaction_group` tools, drop item arguments), OpenAPI at `/api/docs`.
- `resources/ts`: `api/transactions.ts`, `TransactionFormDialog.tsx` (the items editor becomes the lines editor) and the list.
- `internal/legacy/columns.go`: check only.

## Tests

Write them first (failing).

- Migration: build a v1.0-shaped database by running `0001` only, insert one of each: income, expense, transfer (same currency, cross-currency, and from an account to itself), an import row with `dedup_hash`, a cross-space mirror, a debt row with a debt account in another currency, a pending and a confirmed recurring row (estimated), a row with items, a row with amount 0, an uncategorised row, tags. Run `0002`. Check: every account balance, every debt account's progress (`SumDebtPayments`) and every per-currency report total before and after; ids and tags kept; item names in the note; no line of 0; no id reused after deleting the newest row; `PRAGMA foreign_key_check` is clean; every `transfer_in`, `transfer_out` and `debt_*` row has a lock and an owner row.
- Existing `migrate` tests (`TestTablesAreStrict`, `TestForeignKeysAreIndexed`) must pass.
- Ledger: sign, net sign and category type for every transaction type; one payment per account; line currency rule; a transfer between currencies with a fee line; the soft check; the permission matrix as a table test.
- Automation: each condition field and `set_category` on a transaction with several payments and lines.
- Recurring: generating an occurrence twice creates one transaction; editing a pending date does not move the next due date; confirming creates the next occurrence one step later; deleting the latest occurrences does not regenerate them; `nextRunDate` derived after the migration equals the old `next_run_date` for every rule; a template accepts exactly what "create transaction" accepts; an account used by a template cannot be deleted; deleting a category or tag updates templates; a broken template stops its rule with a reason in `error`, both when generating and in the startup job (a template edited directly in the database is caught at start); saving the rule again clears `error`; an estimated payment shows as estimated while pending; a NULL payment amount balances the rest at that day's rates (transfer, and USD line paid from RSD); two NULL amounts are refused.
- Import: the same file twice creates no duplicates; deleting a transaction allows a re-import.
- Benchmark as described under Performance.

## Docs and housekeeping

- Rewrite `docs/split-payments.md` (it describes legs and `group_uuid`), and update `docs/api.md`, `docs/mcp.md`, `docs/spaces.md` (the link table) and `docs/development.md` if it shows the schema. Release notes: items become a note, quantities and prices of v1.0 items are dropped.
- If `internal/migrate` gets an `AGENTS.md`, it must say 0001 is frozen. Create or change it through the `edit-agents-md` skill, after asking (root `AGENTS.md`).

## Open

1. **Debts.** The `debt_*` rows get one payment on the real account, a lock and a `debt_links` row (debt account, amount in its currency), so no data is lost. Whether the debts work turns `debt_links` into a second payment on the debt account, and with which sign, is decided then; it is a migration from data that is all there.
2. **Tolerance of the soft check**: decided during implementation. Ideally the tolerance does not exist, and the check only reports the difference.
3. **Report currency**: today's rate or historical. See [schema-v1.1-other.md](schema-v1.1-other.md).
4. **Refunds**: an income cannot use an expense category, so a refund does not reduce a budget. Left for later.
