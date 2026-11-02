# API

Connect scripts and external apps with an API token.

1. Open **Settings → Security → API** and create a token: a name, the spaces it may reach, what it may do in them and an optional expiration. The token is shown once.
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
| Accounts | accounts and their balance history |
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

`POST /transactions/{id}/confirm` takes an optional `date`; for the open occurrence of a recurring rule it may also take `payments` and `lines` with the real amounts. Confirming or skipping an occurrence makes the rule's next one.

### Recurring rules

A rule is `{"template": {...}, "schedule": {...}, "is_active": true}`. The template is a transaction exactly as above, except that the amount of **one** payment may be `null` (and `"estimated": true`): each occurrence then takes whatever balances the others at that day's rates, which covers a transfer between currencies. The schedule is `{"frequency": "monthly", "interval": 1, "day_of_month": 5, "start": "2026-12-05", "end": null}`. A rule with a template that stops working (an account was deleted) stops, with the reason in `error`; saving it again clears the error.

## Updates

`PATCH` changes only the fields it sends: omitted fields keep their values, and an explicit `null` clears an optional field such as a description or an end date. On a transaction, `payments` and `lines` are replaced as a whole when sent.

## Reference

Interactive reference (Swagger UI): `/api/docs`. Raw OpenAPI 3 spec: `/api/openapi.yaml`. Both are served by the API itself, so they work without the frontend. The spec is merged at startup from `internal/httpserver/ui/base.yaml` and the `openapi.yaml` that each handler package embeds next to its routes.
