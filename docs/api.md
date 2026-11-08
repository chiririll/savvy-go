# API

Connect scripts and external apps with an API token.

1. Open **Account → API tokens** and create a token: a name, the spaces it may reach, what it may do in them and an optional expiration. The token is shown once.
2. Send it as a bearer token.

Creating a token, or giving one more, asks for your password again (an authenticator or recovery code if you use 2FA). So does allowing an AI app to connect, adding or removing a passkey and turning on 2FA. This way someone who gets hold of a signed-in browser cannot add a way in that outlives it. Accounts that sign in only through single sign-on have nothing to confirm with and are not asked. Wrong answers count against the same limit as sign-in attempts.

Finances belong to a space, so their endpoints are under `/api/spaces/{id}/…`. List your spaces to find the id (each has a `name`, an `emoji` and a `color`, the last two `null` when not set):

```bash
curl -H "Authorization: Bearer svy_xxxxxxxx" https://your-savvy.example/api/spaces

curl -H "Authorization: Bearer svy_xxxxxxxx" https://your-savvy.example/api/spaces/1/accounts

curl -X POST https://your-savvy.example/api/spaces/1/tags \
  -H "Authorization: Bearer svy_xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{"name": "from-script"}'
```

## Access

A token acts as its owner but only as far as you let it. Give each token the least it needs; a token made for one script should not be able to read your whole ledger.

**Spaces.** A token reaches the spaces you choose, or all your spaces, including ones you join later. In any other space it answers `404`, the same as a space that does not exist, and `GET /api/spaces` lists only the spaces the token reaches. If a chosen space is deleted, the token loses it; a token left with no spaces reaches nothing.

**Presets.** The quickest way is to start from a role:

| Preset | The token can |
|--------|---------------|
| Viewer | read everything in its spaces |
| Editor | read everything; edit transactions, accounts, budgets, debts, recurring payments, tags and automation (not categories, currencies, reports, transfers or the space). Tick **Allow deleting transactions** to let it delete transactions too |
| Admin | read, change and delete everything in its spaces. Space administration, backups, invitations and security settings stay with you |
| Custom | choose a level for each part below |

**Permissions.** You grant each part of a space separately, as `read`, `write` (add and change) or `delete`. Each level includes the ones before it, so a token can add data without being able to remove it. A request outside the grants answers `403`.

| Part | Covers |
|------|--------|
| Accounts | accounts, their balance history and reconciliation points |
| Transactions | transactions, summaries, imports and uploads |
| Categories, Tags | categories and tags |
| Currencies | the space's currencies and rates |
| Budgets, Debts, Recurring, Automation | budgets, debts, recurring transactions, automation rules and their logs |
| Reports | all reports. They aggregate your transactions, so a reports grant shows their totals even without a transactions grant |
| Space | the space itself, its members, settings and links |
| Transfers | transfers with linked spaces |

From a script, the body is `{"name": "Budget bot", "preset": "editor", "space_ids": [1]}` or, with a custom set, `{"name": "Budget bot", "permissions": ["transactions:read", "budgets:read"], "space_ids": [1]}`; see the interactive reference.

**A token is never more powerful than you.** Your role in each space is checked on every request, so an Admin token in a space where you are a viewer still cannot write, and if you are demoted or removed, your tokens lose that access at once.

**Synced spaces.** Categories and tags of spaces in a [sync group](spaces.md) are mirrored into the group's other spaces. Changing them with a token needs a token that reaches every space of the group, and you must be allowed to edit each of them; otherwise the request answers `403` and nothing is changed. Tokens cannot create or change sync groups. The token form warns you when the spaces you chose are in a sync group, and names the spaces the token would not reach. Tokens cannot manage users, identity providers, passwords, 2FA, passkeys, API tokens, settings, backups, invitations or space administration, whatever they hold. Revoke a token in the same screen to cut an app off at once. To give an app more or less later, press the pencil next to the token: you can change its spaces and what it may do without creating a new token, and the app keeps its secret. A change applies from the app's next request.

## Reconciliation points

A reconciliation point pins an account's balance to the real one at the end of a date. They are part of the accounts permission (`accounts:read` to list, `accounts:write` to add, `accounts:delete` to delete). Amounts are JSON numbers in the account's currency; `differenceBase` is in the base currency, `null` when the currency has no rate.

```bash
curl -X POST https://your-savvy.example/api/spaces/1/reconciliations \
  -H "Authorization: Bearer svy_xxxxxxxx" -H "Content-Type: application/json" \
  -d '{"date": "2026-11-01", "items": [{"accountId": 1, "actualBalance": 950.5, "note": "counted"}]}'
```

It answers `201` with the points, in the order sent: `{"data": [{"id": 5, "accountId": 1, "date": "2026-11-01", "actualBalance": 950.5, "expectedBalance": 1000, "difference": -49.5, "note": "counted", "differenceBase": -49.5}]}`. All items are stored or none. The same account and date again replaces the point. A `422` means an empty list, a bad or future date, an account twice, an unknown, debt or space account, or a balance with more decimals than the currency.

- `GET /reconciliations?account_ids=1,2&start_date=&end_date=` lists points, newest first.
- `GET /accounts/{id}/reconciliations` lists one account's points.
- `DELETE /reconciliations/{id}` removes a point (`204`).
- `GET /reconciliations-days?start_date=&end_date=` (both required) groups points by day: `[{"date": "2026-11-01", "differenceBase": -49.5, "unconverted": false, "points": [...]}]`. `unconverted` is true when a point without a rate is left out of the sum.

## Transactions

A transaction is one event with the **payments** that moved the money and the **lines** that say what it was for.

```json
{
  "type": "expense",
  "date": "2026-11-02",
  "description": "Souvenirs",
  "payments": [
    {"account_id": 1, "amount": "-50.00"},
    {"account_id": 4, "amount": "11.50"}
  ],
  "lines": [
    {"category_id": 7, "currency_id": 2, "amount": "40.00", "note": "magnets"}
  ],
  "tag_ids": [3]
}
```

- A payment is money that moved on one account, signed, in that account's currency: an expense takes money (negative), an income brings it (positive), and change is a payment of the opposite sign on another account, often in another currency. An account appears once. A payment of zero is refused.
- A line is an amount for a category: positive, in one of the currencies of the payment accounts (`currency_id` can be left out when they share one), in a category of the transaction's type. An income or expense needs at least one line, and a transfer may have fee lines in expense categories. Reports read lines only, so change is never income.
- A line's `note` is markdown (bold, italic, lists, task lists, links, inline code); a list of products is a markdown list. It has no length limit of its own. Upgrading from v1.0 turns a transaction's item names into such a list; quantities and prices are dropped.
- The database does not tie payments to lines. When you save, the API converts both to the base currency at the current rates and answers `difference` (payments less lines) if they disagree. It never refuses the save for that.
- Debt and space accounts cannot be paid directly: a debt moves through the debts endpoints, a space account through a transfer to another space.
- Keys the API does not know are refused, so a client sending the flat shape of earlier versions (`account_id`, `amount`, `items`) gets a `422`.

An answer carries `payments`, `lines`, `amount` (what the payments net to, `null` when they span several currencies), `actions` (`edit`, `editDetails`, `delete`, `duplicate`, `confirm`, `skip`) and `restrictions`, the reasons a transaction cannot be changed freely:

| Reason | Meaning | Allowed |
|---|---|---|
| `debt` | a payment on a debt account | details only (description, tags, notes) |
| `space_transfer` | one side of a transfer between spaces | everything except its space payment, the paired payment and the date |
| `transfer_review` | that transfer is in review | nothing |
| `recurring_pending` | the open occurrence of a recurring rule | confirm or skip |

`POST /transactions/bulk` takes `{"transactions": [...]}`, up to 500 records shaped like one `POST /transactions` body, and stores all of them or none in one save; an invalid record fails the request with its position (`transaction 3: …`). It answers with the created transactions in the order sent.

`PATCH /transactions/bulk` takes `{"transactions": [{"id": 12, ...}, ...]}`, up to 500 items, each a transaction id and the fields to change as for one `PATCH /transactions/{id}`; what an item leaves out is kept. All items change or none do, in one save. A bad item fails the request with its position (`transaction 3: …`), an unknown id answers 404, and an id listed twice is refused. It answers with the updated transactions in the order sent. The audit entries of one call share a `batch_id`, so one rollback undoes it all.

### External id and duplicates

Send `external_id` (up to 255 characters) with a new transaction to give it your own key, for example the receipt number `F6HDVMNQ-5105`. Savvy records where the request came from as `externalSource`: `ui` for the app, `api` for a personal API token, `mcp` for a connected AI assistant (the CSV import writes `csv`). You cannot set the source, so nobody can pose as another one. The key can be set only when the transaction is created; an update that sends one is refused.

- **Same source, same id: 409.** Creating it again, for example a retry after a timeout, stores nothing and answers `{"error": "external_id_exists", "external_id": "…", "transaction_id": 42}`, with the id of the transaction that has the key. A bulk request is all or none, so one repeated key stops the whole request, and a key repeated inside the request counts too.
- **Same id, another source: stored, with a warning.** The answer lists such transactions in `externalMatches` (`[{"source": "api", "transactionId": 42}]`). It may be the same event entered twice or two events that share a number; look before you delete one.
- **"Was this receipt entered?"** `GET /transactions?external_id=F6HDVMNQ-5105` finds it under any source; add `external_source` to narrow it to one.

`POST /transactions/find-similar` helps when a source has no key, such as a bank statement. It takes `{"items": [...], "date_tolerance_days": 1, "max_per_item": 5}` with up to 500 items. An item is `{"key", "date", "amount", "currency" or "account_id", "description", "external_id"}`, and the answer has one entry per item, in order, with `matches`: the entered transactions that look like it, best first, each with its `reasons` (`external_id`, `amount`, `date`, `account`, `description`). The amount must equal a payment exactly in the item's currency, the sign is ignored, and an amount with more decimals than the currency keeps matches nothing, so 10 USD never matches 10 JPY. The date may differ by `date_tolerance_days` (0 to 31). Skipped transactions never match. A match is a hint, not proof. The request stores nothing, but it is a POST, so it needs write permission.

`POST /transactions/{id}/confirm` takes an optional `date`; for the open occurrence of a recurring rule it may also take `payments` and `lines` with the real amounts. Confirming or skipping an occurrence makes the rule's next one.

### Recurring rules

A rule is `{"template": {...}, "schedule": {...}, "is_active": true}`. The template is a transaction exactly as above, except that the amount of **one** payment may be `null` (and `"estimated": true`): each occurrence then takes whatever balances the others at that day's rates, which covers a transfer between currencies. The schedule is `{"frequency": "monthly", "interval": 1, "day_of_month": 5, "start": "2026-12-05", "end": null}`. A rule with a template that stops working (an account was deleted) stops, with the reason in `error`; saving it again clears the error.

## Audit log

Every change to a transaction (created, edited, deleted, confirmed, skipped, or part of a bulk create) is written to the space's audit log in the same save as the change: when, who (the user, and the API token if there was one), through which door (`ui`, `api`, `mcp`, or `system` for rules and the like, with no user), and the transaction before and after, with exact amounts as minor units. Occurrences that recurring rules make on schedule are logged as `system`. A CSV import is one entry (`entity_type` `import`, no entity id) saying how many transactions it created, skipped and rejected; it is not rolled back from the log, and its transactions can be found by the external source `csv`. Not logged yet: accounts, categories and the other parts of a space, and putting transactions in a different order within a day.

| Route | Needs | Does |
|-------|-------|------|
| `GET /spaces/{id}/audit-log` | `audit:read` | newest first; filters `entity_type`, `entity_id`, `op`, `source`, `before` (the next page), `limit` (up to 200). A token sees the entries it made, a space admin all (and may filter with `token_id`), any other member their own. Each entry has `rollbackable` and, if not, `rollbackBlockedReason` |
| `GET /spaces/{id}/audit-log/{entry}` | `audit:read` | one entry with `before` and `after` |
| `POST /spaces/{id}/audit-log/rollback` | `audit:write` | undo `{"ids": [..]}` or a bulk call `{"batch_id": ".."}`, newest first, all or none. A token undoes only its own entries; in the app a space admin undoes any |
| `DELETE /spaces/{id}/audit-log` | space admin, signed in | clear the log (`?token_id=` only that token's entries) |

Rolling back puts the saved copy back; it does not repeat the change through the normal rules, so a token that may add but not delete still takes its own create back. It answers `409` when the transaction was changed after the entry (`changed_since`), the entry was already rolled back (`already_rolled_back`), or a restored `external_id` is taken again (`external_id_exists`). The `audit` permission is separate from the others: an Editor token has `audit:write`, a token made before it existed does not.

**Clearing** is manual and only by a space admin in a signed-in session; nothing prunes the log by itself. Afterwards the log holds one entry saying who cleared it, when, and how many entries went. Cleared entries can no longer be rolled back. Entries stay after the transaction, the token or the user they name is deleted.

The app shows the log under **Space → Recent actions** (see [spaces](spaces.md#recent-actions)). `GET /users` and `GET /spaces/{id}/members` carry `lastActiveAt`: when a session or API token of the user was last used, `null` if never.

## Updates

`PATCH` changes only the fields it sends: omitted fields keep their values, and an explicit `null` clears an optional field such as a description or an end date. On a transaction, `payments` and `lines` are replaced as a whole when sent.

## Reference

Interactive reference (Swagger UI): `/api/docs`. Raw OpenAPI 3 spec: `/api/openapi.yaml`. Both are served by the API itself, so they work without the frontend. The spec is merged at startup from `internal/httpserver/ui/base.yaml` and the `openapi.yaml` that each handler package embeds next to its routes.
