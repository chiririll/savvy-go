# Backlog after two days of real use

Status: raw notes from 25-28 Oct 2026, sorted by area. Not prioritised or scheduled. Checked against the code on 29 Oct 2026 (branch `releases/v1.1`); items that say "not checked" or "not traced" were not read in the code. Source: entering real transactions through Claude (MCP) and using the web app.

Priority hints: **P1** blocks or corrupts data, **P2** daily friction, **P3** nice to have. Items marked "see plan" are probably solved by [payments-split.md](payments-split.md); check there before designing anything new.

Versioning rule: the next two releases are v1.0.2 (minor) and v1.1 (major). Only v1.1 may change the schema: `0002_transactions.sql` is unreleased (v1.0.1 is the latest tag) and being rewritten for the transactions rework, so migrations go into it. v1.0.2 has no migration, so it takes only items that need none and do not depend on the rework. Items are tracked as GitHub issues with milestones v1.0.2 and v1.1; this file keeps the context and the open questions.

## Bugs

| P | Item | Notes |
|---|---|---|
| P2 | `update_transaction` with `is_estimated: true` returns `isEstimated: false` | Probably not a lost write: `keepsEstimate` in `internal/ledger/transaction.go` keeps the flag only for a **pending** income, expense or transfer **without items**; for anything else it is silently set to false. Both records ("airport", "rent change") were probably confirmed. Claude wanted the flag on confirmed transactions. Either say in the MCP tool description that it applies only to pending ones, or let confirmed transactions be estimated too. In both cases the flag must not be dropped silently. Confirm with a failing test first. In the split plan the flag moves to `payments`. |
| P1 | Dashboard crashes when the date range moves from 2026 to 2025: `can't access property "length", s is null` | Likely cause: report code declares empty results as `var out []T` (for example `internal/reports/calc.go`), which Go sends as JSON `null`, and the frontend reads `.length` on it (for example `summary?.data.length` in `ExpensesByCategoryCard.tsx`). Fix both sides: return `[]`, and guard in the card. Reproduce with a story or test using `data: null`. |
| P2 | Receipts page jumps after a transaction is edited | Not traced in code. Scroll or layout reset after the refetch. |
| P2 | Accounts in the top panel do not update after the initial balance changes | Not confirmed by the code: `useUpdateAccount` invalidates the whole `['accounts']` prefix, which covers the balance summary. Find which query the top panel reads, and check whether the server returns the old balance. |
| P2 | Cancel button is pushed out of the modal for creating a recurring transaction | `RecurringFormDialog` uses the shared `FormDialog`, and its footer does not use the footer-start slot, so the cause is not obvious. Find the width where it happens (desktop dialog or mobile sheet) first. The story only clicks Save: add a check that Cancel is visible. |
| P1 | Email stays unconfirmed after the password was set through the emailed link | `passwordAccept` in `internal/httpserver/auth/session/session.go` sets the password and never marks the email as confirmed. The link is created by an admin (`internal/httpserver/admin/users/users.go`), who sometimes gets the link back instead of having it emailed. Holding that link does not prove the mailbox, so confirm only when the server emailed it: the token needs to remember that. Failing test first. |

## Transaction list

Today the list filters by type, date range, categories and tags (`TransactionFilterDetails.tsx`).

- Filter by account (P2). The API and the MCP tool already take `account_id`, and `TransactionFilters` has the field; only the filter panel lacks the control.
- Filter by currency (P2). Missing in the API too (`internal/db/filter/filter.go`). Combine with the account filter.
- Show weekdays next to dates (P3). Rows show a relative day (`relativeDay`).
- Free-text search over description, amount and item names ("armchair", "everything for 1 879") (P2). No search parameter exists in the API.
- Group display: collapse a group to one row with a total, expand on click. See plan. Today each part of a split payment is its own row with a "Part of a payment" badge (`groupUuid`).

## Dashboard, reports, budgets

- "Largest transactions" must follow the selected period, not a fixed last 30 days (P2). Confirmed: `pages/dashboard.tsx` pins it to `last_30_days` on purpose. Budgets: not checked.
- Arrows to shift the period by its own span (previous or next month, week, and so on) (P2). The dashboard `PeriodPicker` and the reports `PeriodSelector` have no arrows.
- Choose the currency of a report (P2). Reports are summed in the base currency using each currency's rate (`internal/db/filter/filter.go`, `get_report` says so). Multi-currency is the core feature, so a choice of report currency should convert with stored rates, not hide other currencies.
- Expense dynamics chart in reports:
  - the series are chosen in a popover (`SeriesPicker` in `DynamicsControls.tsx`), which is the "combobox"; replace it with the "hide line" legend used on the dashboard;
  - on hover over a day, list only non-zero expenses.
- Money flow chart (a Sankey diagram on the reports overview, also the MCP `money_flow` report): decide what it shows. It is judged useless; either explain it in the UI or remove it (P3).
- Compare months by category (P3).
- Average item price from item history, per shop ("milk at Lidl vs Maxi") (P3).
- Balance chart by day with dips below zero highlighted (P2). Needed to find the gaps on 24 and 26 August. A net worth history report and `NetWorthChart` already exist, and the accounts API has `balance-history`; check what they show per account before adding a new chart.
- Subcategories, so charts read better: a parent category groups its children visually (P2). Reports must roll children up into the parent and allow drilling down. Today `categories` has no parent column, so this needs a migration. See the versioning rule at the top. Also decide: names are unique per type, so can two parents each have "Other"? Sync groups copy categories between spaces, so the parent link must sync too. A budget on a parent should cover its children.
- Click a donut segment in reports to show the transactions of that category (P2). Pairs with subcategories: a click on a parent includes its children.
- Two-way budgets (to think about): two scales facing each other, for example groceries and cafe. Unspent groceries budget can be spent on cafe. Needs a rule for the direction and the limit of the transfer, and for budgets in different currencies.
- Budget instalments (to think about): an expense that does not fit the current month can be spread over several months for budget purposes only. An expense of 200% of the budget split over 4 months counts 25% in each of the next 4 months. The transaction itself and the account balance do not change; only the budget view does.

## Balances and reconciliation

Most painful part of the two days: finding the gap meant exporting everything and summing by hand. None of the three exists in the code: `list_accounts` returns only the current balance, and there is no reconciliation record.

1. Balance of an account at a date and time: `get_balance(account_id, at)` and `get_balance_history(account_id, from, to)` with one balance per day (P1).
2. Reconciliation as its own record type: "04.09, actual 5 586". The app stores these points and shows the difference to the computed balance at each one, so a gap localises itself. API: `reconcile_account(account_id, date, actual_balance)` creates the adjustment with a clear note (P1).
3. Count by denominations: "5 490 in notes + 96 in coins", a separate 50 € note (P3).

## Groups, receipts, items

All of this overlaps the payments/transactions split. Check coverage in [payments-split.md](payments-split.md). What exists today: split payments as transactions sharing a `group_uuid` ([split-payments.md](../split-payments.md)), and receipt lines (`transaction_items`) that belong to one transaction with one category. Items are not allowed on split payments.

- A receipt is one group with a shared total and a per-category breakdown. One Lidl receipt is now three transactions ("groceries", "household", "shampoo").
- Category per item, not only per transaction. One receipt, one record (P2).
- An event or trip as a group: "Belgrade 02.10" (trains, snacks, printing), "Residence permit legalisation" (fees, transfers, notary). Shows the total cost of the event across categories.
- Linked operations: paid 200 €, got back 50 € in cash and 4 640 RSD. This is one "payment with change", not two unrelated rows. Also "exchange plus the purchase it was made for".
- Receipt total versus the amount actually paid (196 → 200).
- Quick entry without a receipt: "bus 100", "donation 200" in one line, like writing in a chat. Otherwise 3-6k of "unaccounted" builds up (P2).

## Import

Today only CSV import exists (`internal/importer`, `CsvImportWizard`). It avoids duplicates with a `dedup_hash` per account, not with an id from the source.

- Import by fiscal receipt link or QR code: `import_receipt(url)`. The Savvy server downloads the JSON (for example from `suf.purs.gov.rs`) and fills items, shop, time and payment method (cash or card). The remote Claude environment could not reach the tax site, so it must run server-side (P2).
- `external_id` or idempotency key, for example the receipt number `F6HDVMNQ-5105`. A retry after a timeout then creates no duplicate, and "was this receipt entered?" is one query (P1). Related: `import_fingerprints` in the split plan.

## MCP / AI access

- Batch create: `create_transactions` with an array. 133 single calls were slow and the server failed to answer twice (P1). Not in the code; `create_transaction` takes one record.
- Compact `list_transactions`: choose fields (P1). Not in the code. Filtering by `account_id` already works, so only the fields choice is missing. Today every record repeats the full account object with its currency, and a month of data was cut by the response limit (pages of up to 100).
- Rollback endpoint for MCP (P2). Needs a design: roll back what, to which point (for example a batch id). Nothing exists.
- OAuth client choice when connecting from Claude. Today only Dynamic Client Registration works, and only while an admin has opened a 5-minute window in **Administration → Security** (`docs/mcp.md`). The server should support all three modes the client offers:
  - Client ID Metadata Document (CIMD): Claude's identity is read from a URL hosted by Anthropic, nothing to set up, but the server must support it;
  - Dynamic Client Registration (DCR): works with most servers, but creates many client registrations on a busy one;
  - own OAuth client: user enters a client ID, secret only if the authorization server requires one.
- Claude app shows a YouTube icon for the Savvy connector. Needs a proper icon from the server (favicon or CIMD logo) (P3). The server publishes no icon or logo in its metadata today.

## Shared spaces

- Who paid and for whom (you, a partner, shared), plus a monthly "who owes whom" balance. The tags for two people already hint at this (P3).
- Payments for a transaction from another space (P2). See `docs/spaces.md` for signed cross-space transfers; the split plan has `space_transfer_payments`.

## Instance access

- Create a user in the admin panel without entering their email (P2). Today `email` is required there (`internal/httpserver/admin/users/users.go` validates it, and `users.email` is `NOT NULL UNIQUE`), so an admin must know the address before the person exists. Wanted: the admin makes an invite link, and whoever opens it enters their own email and password. Needs expiry, a use limit and revocation.
  - Space invitations already work by link without an address (`docs/spaces.md`, Invitations), but they belong to a space. This is the same idea at the instance level, with no space.
  - Option A, an instance invitation row (token hashed, role, expiry, optional note), registered through its own link like a space invitation. No change to `users`. Recommended.
  - Option B, a user row without an email. It needs a server migration that rebuilds `users` to drop `NOT NULL`, and every place that assumes an email exists (login, mail, SSO matching) must handle a missing one.
  - If the server requires email confirmation, the new account confirms its own address after registering.

## Backups and distribution

- Backups to Google Drive (P3). Backups are only local files and downloads today (`docs/data-and-backups.md`).
- Show the latest version in the package repository (P3). The repository page is `deploy/repo/index.html`, built by `deploy/repo/build.sh`.

## Draft issues (not created on GitHub yet)

### Monitoring: CPU, memory and load average empty on a VPS (milestone v1.0.2)

**Problem.** On the VPS the CPU gauge, load average and memory gauge show "unavailable". Storage and process memory work. The cause is in `resources` (`internal/httpserver/admin/monitoring/monitoring.go`): `cpu.load`, `cpu.load_percent` and the whole `memory` object are hardcoded `null`. Only Go runtime figures are filled. The frontend already handles real values (1/5/15 load, memory source `cgroup` or host).

**Proposal.** Add a `Collector` interface in `internal/httpserver/admin/sysinfo` (`CPU`, `Memory`), picked by host:

- Linux: `/proc/loadavg`, `/proc/meminfo`, cgroup v2 (v1 fallback) for the container limit.
- Windows and other: leave the fields `nil`, as today.

A value that cannot be read stays `nil`, so the API shape and the SPA do not change. CPU percent is load divided by cores, with no background sampling. No migration.

**Done when.** On a Linux VPS, with or without Docker, load average, CPU and memory show real numbers.


## Open questions

- Money flow chart: keep, explain or remove?
- MCP rollback: what is the unit of rollback?
- Report currency: convert at the rate of the transaction date or of today?
- Two-way budgets and budget instalments: how do they work with several currencies?- Which of the group types (receipt, event, linked operation) stay in the model after the split, and which become tags?
