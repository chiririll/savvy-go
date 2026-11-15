# Plan: account passthrough to a linked space

Status: idea, discussed 15 Nov 2026. Nothing is built. Items marked **proposed** wait for the owner's decision.

Closes #106 (payments for a transaction from another space). #105 is covered by debts and is not part of this plan.

## Goal

An account of one space can be **shared** into a linked space. There it is picked like any other account. A transaction paid with it takes the money from the real account, while the category, the budget and the reports stay in the space where the transaction was entered.

Example: the family space has an expense "Groceries 80 EUR" paid with "Card" of the personal space. The personal card goes down by 80 EUR. The family space counts 80 EUR of groceries.

## Terms

- **Home space**: the space the real account belongs to.
- **Receiving space**: the linked space the account is shared into.
- **Shared account**: the row in the receiving space that stands for the real account.

## Checked in code (15 Nov 2026)

- A transfer between spaces is one signed record in both spaces (`space_transfers`) plus one transaction per side. In each space a **space account** of type `space` stands for all other spaces, one per currency (`accounts_space_currency_unique` in `internal/migrate/sql/space/0001_base.sql`). The payment on it carries `space_transfer_uuid`.
- `internal/transfers` writes its side transactions itself through `db/repo/transfers` (`SaveMirror`). It does not use `internal/ledger`, and the ledger does not use it.
- The ledger already allows an `expense` with one payment on a space account when the caller is the owner of space transfers (`checkServiceAccount` in `internal/ledger/shape.go`). `frozenParts` in `transaction_compare.go` already keeps that payment and the date fixed for other callers. Nothing calls `Transactions.As(ReasonSpaceTransfer)` yet.
- The signed payload stores each side's account as an opaque number (`Side.AccountID`). The other space never reads it.
- v1.1 is released, so the schema change is a new additive migration, `space/0003`.
- An account with payments cannot be deleted (`Accounts.Delete`).
- An API token reaches only the spaces it was given (`httpx/guard/space.go`). Whether a transfer checks the token for the **other** space is not verified, see the open questions.

## Design

### The shared account

- A row in the receiving space's `accounts` with type `space`, the currency of the real account and two new columns: the home space's uuid and the real account's id.
- One row per shared account, not the per-currency space account. Reasons: two shared accounts of one currency can pay the same transaction (a payment is unique per transaction and account), the transaction list filters by it, and its balance is what that account paid into the space.
- Its name is a copy, `<account name> · <space name>`. It is refreshed after the real account or the home space is renamed, and at the startup sync.
- If the receiving space lacks the currency, sharing adds it with the rate from the currency data. A currency without a known rate is refused, as in space creation.
- The row is the only record that the account is shared. No server table. The home space finds its shares by reading its linked spaces.

### Schema, `space/0003`

```sql
ALTER TABLE accounts ADD COLUMN remote_space TEXT;
ALTER TABLE accounts ADD COLUMN remote_account INTEGER;
DROP INDEX accounts_space_currency_unique;
CREATE UNIQUE INDEX accounts_space_currency_unique ON accounts (currency_id)
    WHERE type = 'space' AND remote_space IS NULL;
CREATE UNIQUE INDEX accounts_remote_unique ON accounts (remote_space, remote_account)
    WHERE remote_space IS NOT NULL;
```

No table rebuild. The ledger checks that both columns are set together and only on type `space`. `GetSpaceAccountID` must add `remote_space IS NULL`.

### The transfer record

Unchanged: same payload, same signatures, no new `sig_v`.

| Transaction in the receiving space | Record | Home space gets |
|---|---|---|
| Expense paid with a shared account | from the real account to the shared account | `transfer_out` |
| Income received on a shared account (a refund) | from the shared account to the real account | `transfer_in` |

Both sides have the same amount and the same currency. No rate is used or stored.

In the receiving space there is no side transaction. The payment on the shared account, inside the user's own transaction, carries the uuid.

### Writing

Neither `ledger` nor `transfers` may import the other, so a new package above both, `internal/passthrough`, runs a save in one `store.InSpaces`:

1. `transfers` signs and stores the record in both spaces and writes the home space's side transaction.
2. `ledger`, as the owner of space transfers, writes the receiving space's transaction with the uuid on the payment.

The record goes first because `payments.space_transfer_uuid` references it.

- **Create**: one record per shared account in the transaction.
- **Update**: a change of the amount, the date or the account of a shared payment writes the next version of its record. A shared payment that is removed deletes its record (a tombstone). A change of lines, tags or the description touches the receiving space only.
- **Delete**: every record of the transaction gets a tombstone.

The transactions API sends a save through `passthrough` when the input or the stored transaction has a payment on a shared account. Everything else goes to the ledger as today.

`transfers` needs variants of `Create` and `change` that run inside a caller's `InSpaces`. `writeRecord` must not write a side transaction when its own side is a shared account.

### Ledger rules

- A shared account may be paid in `expense` and `income`. Today only `expense` is allowed.
- A transaction may have several shared payments. Today the limit is one space payment.
- A transfer between a shared account and an own account stays what it is today: a transfer between spaces. The form sends it to the transfers API.
- Refused in the first version: recurring rules, automation actions and CSV import on a shared account, reconciliation points, and a transfer between two shared accounts.

### Rights and privacy

- **Share and stop sharing**: an admin of the home space (**proposed**). The link already needed an admin of both spaces.
- **Pay with a shared account, change or delete that payment**: an editor of both spaces, the same rule as a transfer (**proposed**).
- A member of the receiving space who is not in the home space sees the transaction and the name of the shared account. They can change the description, the tags and the categories and notes of lines. They cannot change the money or the date, and they do not see the real balance.
- An API token and the MCP server must reach both spaces to pay with a shared account.

### Balances and reports

- The home space shows the movement as a transfer to the other space. It is not spending there.
- The receiving space counts the lines in budgets and reports as for any transaction.
- A shared account is not part of the receiving space's net worth. Space accounts are already left out.
- The stored balance of a shared account is what it paid in, as a negative number. The UI does not show it as a balance, see below.

### Sync, restore and unlink

The existing rules stay. The new cases:

- **A record the receiving space lacks is accepted** (it was restored from an older backup): an expense, or an income, is made with one line in the default category. It is pending until accepted, as today.
- **A tombstone arrives in the receiving space**: the payment is removed. A transaction left with no payment is deleted.
- **A changed amount or date arrives**: the payment and the date follow. The lines stay, so the form may warn that they cover too little or too much.
- **Unlinked, or the home space is not on this server**: the shared account is not offered in pickers. Its transactions stay and are frozen, like transfers.
- **The real account is inactive or missing**: the shared account is not offered. History stays.

`transfers` writes these through `db/repo/transfers`, as it writes side transactions today, so it still does not use the ledger.

## API and MCP

- `GET /spaces/{id}/shared-accounts`: what the space shares out and what is shared into it.
- `POST /spaces/{id}/shared-accounts` with `account_id` and `target_space_id`; `DELETE /spaces/{id}/shared-accounts/{target}/{account}`. Admin routes of the home space.
- `GET /accounts` returns shared accounts with `type: "space"` and a new `shared` object: the home space (id and name), `available` (linked, both here, the real account active), `canPay` (the caller is an editor of both), and `balance`, present only when the caller is a member of the home space. A new filter leaves them in the list a picker asks for.
- `POST`, `PATCH` and `DELETE /transactions` take a shared account in `payments` like any account. No new endpoint.
- A transaction's `restrictions` name the reason when the caller cannot change the money.
- MCP: `list_accounts` returns shared accounts. `create_transaction` and `update_transaction` accept them when the token reaches both spaces.

Names are provisional.

## UI

### Sharing an account

- **Space → Settings → Linked spaces** (`components/features/spaces/settings/Links.tsx`): each linked space becomes a fold. Inside, two lists: **Shared from this space** with a **Share an account** button that opens a picker of this space's accounts, and **Shared into this space**, read-only. Each row has **Stop sharing**, which asks first and says that past transactions stay.
- **Account form**: a read-only line "Shared to: Family" that links to the settings section. No second place to manage it.

### Accounts page

- A separate group **From other spaces** under the own accounts. A row shows the name, `EUR · <home space>` and the space's emoji badge.
- The amount is the real balance for a member of the home space, and empty for everyone else.
- No edit, delete, reorder or reconcile. A shared account that is not available is greyed out with the reason (unlinked, account inactive).

### Transaction form

- `AccountSelect` lists shared accounts after the own ones, under a heading with the home space's name and badge. They appear in the payment rows of an expense and an income, and in the change rows.
- A member who cannot pay with a shared account sees it disabled with a tooltip: "You need to be an editor in <space>".
- The balance hint under the account (`TransactionBalanceHint`) shows the real balance before and after for a member of the home space, and nothing for others. The warning before an account goes below zero covers shared accounts too.
- Split payments work as today: own and shared accounts can be mixed in one transaction.
- The square tab for a transfer to a linked space stays. In the **Transfer** tab a shared account can also be picked as source or destination; the form then saves through the transfers API.

### Transaction list and details

- The list row is unchanged. The details dialog shows the payment as `<account> · <space badge>`.
- A member without rights in the home space gets the existing short form ("Only the details") with the text: "Paid from an account of <space>. Its amount and date can be changed by an editor of both spaces."
- A transaction whose transfer is in review does not open, as today.

### Home space

- The movement appears as today's transfer to another space. Its description is the transaction's title in the receiving space.
- **proposed**: clicking it offers "Open in <space>" for a member of both.

### Strings

New keys in `locales/en` and `locales/ru`. `ACCOUNT_TYPE_CONFIG.space` keeps its icon; the label of a shared account is its own name, not "Other space".

## Order of work

Each step ends with its tests green. Failing test first.

1. **Schema and repository.** `space/0003`, `sqlc`, repo methods to make, list and switch off a shared account, `GetSpaceAccountID` fixed. Tests in `db/repo` with `db/dbtest`, and a migration test.
2. **Sharing.** `internal/passthrough`: share, stop sharing, list, name refresh, currency added on demand. Rights. Add the package to `businessPackages` in `store/arch_test.go`.
3. **Transfers.** In-transaction variants of create and change; no side transaction on a shared side; the sync and review cases above. Tests next to `transfers_test.go`: both sides after create, update and delete, a restored receiving space, a restored home space, unlink.
4. **Ledger.** Shape rules (income, several shared payments), the refusals of the first version, `Reasons` and `restrictions` for a member without rights. Table tests in `shape_test.go` and `allowed_test.go`.
5. **Orchestrated save.** `passthrough` create, update and delete in one `InSpaces`, with audit entries in both spaces. Rollback from **Recent actions** goes through the same path or is refused.
6. **API and MCP.** Routes, DTOs, OpenAPI, token checks for the other space, `lib/backend-contract.test.ts`.
7. **Frontend.** Settings section, accounts page group, `AccountSelect`, balance hint, details dialog, short form text, locales. Types with `tsc`, unit tests; stories run on the remote Docker.
8. **Docs.** `docs/spaces.md` (new section), `docs/accounts.md`, `docs/transactions.md`, `docs/split-payments.md`, `docs/api.md`, `docs/mcp.md`, `docs/data-and-backups.md`.

## Open questions

- **Who may pay.** Proposed: an editor of both spaces. The alternative, any editor of the receiving space, writes into the home space without one of its members and breaks the rule that transfers keep today.
- **Who may share.** Proposed: an admin of the home space. Should an admin of the receiving space be able to refuse or remove a shared account?
- **Tokens.** Does a transfer today check that an API token reaches the other space? If not, fix it with this work.
- **Rejecting in review.** A rejection in the home space removes a payment, or a whole transaction, in the receiving space. Is that acceptable, or should the receiving space keep the transaction with the payment moved to a "needs attention" state?
- **Recurring rules.** A rule that pays from a shared account runs with no user. Which rights apply? Left out of the first version.
- **Name of the feature in the UI.** "Shared account" is used here. "Passthrough" stays an internal word.
