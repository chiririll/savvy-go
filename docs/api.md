# API

Connect scripts and external apps with an API token.

1. Open **Settings → User → API** and create a token: a name, the spaces it may reach, what it may do in them and an optional expiration. The token is shown once.
2. Send it as a bearer token.

Finances belong to a space, so their endpoints are under `/api/spaces/{id}/…`. List your spaces to find the id:

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
| Editor | read everything, add and change transactions. Tick **Allow deleting transactions** to let it delete those too |
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

## Updates

`PATCH` changes only the fields it sends: omitted fields keep their values, and an explicit `null` clears an optional field such as a description or an end date.

## Reference

Interactive reference (Swagger UI): `/api/docs`. Raw OpenAPI 3 spec: `/api/openapi.yaml`.
