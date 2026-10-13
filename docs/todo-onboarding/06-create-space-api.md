# 06 Create a filled space in one request

Depends on 02 and 05. Extends `POST /api/spaces`; no new endpoint.

## Request

Optional fields beside `name`:

```
baseCurrency: "EUR"
currencies:   ["USD", "GBP"]                       # codes only
accounts:     [{ name, type?, currency: "EUR", initialBalance? }]
categories:   [{ name, type, icon, color, isDefault }]
tags:         ["Family", ...]
```

## Behavior

- A body with only `name` behaves as today (an empty space), so existing callers and tests keep working.
- `spaces.CreateOwned` first (limits and the guest check apply), then everything is created in one `store.Tx`: the base currency, the other currencies via `FindOrCreateByCode` (so their rates come from the server's rate source), the accounts (amounts in the same JSON number format as `accountsStore`; money never float), the categories, the tags.
- If filling fails, delete the new space again (`spaces.Create` already does this for its own failures; export the same cleanup as `spaces.Discard`) and return the error. Nothing remains: no registry row, no file.
- Unknown rate: the source can return rate zero when it is offline, and `Currencies.Create` turns a zero rate into 1. Decide in the implementation whether to refuse (clear error) or leave the rate to the scheduled update; never silently store 1 for a foreign currency.
- Keep the code in one function in `httpserver` (it needs `spaces` and `ledger` together) so it can be tested directly.
- Update `httpserver/apidocs/openapi.yaml`.

## Tests (first)

- Name only: old behavior.
- Full body: base currency is created first, other currencies get rates, accounts have their balances, the categories (the two defaults always present) and tags exist.
- The space limit and the guest check are enforced.
- A failure in the middle leaves no space (registry and file).
- Unknown rate handled as decided.
