# 09 — Search and a shorter list in the create-space currency pickers

## Problem

The create-space modal picks the base currency and the other currencies from
plain Radix `Select`s that render the whole catalog (~170 items) with no
search. And "sometimes it does not contain needed currencies": the catalog
falls back to a short built-in list when the rates service is down.

## Where

- `resources/ts/components/features/spaces/create/EntitiesStep.tsx` `Currencies`:
  base `Select` (`data-testid="create-space-base"`) with every `options` item,
  and a second picker for the others (`free` list). The account currency
  select in `Accounts` is a third one.
- `resources/ts/components/shared/CurrencySelect.tsx`: already a searchable
  popover (`matchesQuery` on code and name) for existing and catalog
  currencies, used inside a space; the modal does not use it.
- Catalog source: `GET /currencies/catalog` →
  `internal/httpserver/api/currencies/currencies.go` → `ledger.Currencies.Catalog`
  and, for the server-wide one, `internal/currencydata/currencydata.go`
  `Catalog` / `metaList`. When the remote metadata fails, `fallbackMeta()` is
  used and `s.metaUntil = now + Retry`; the result is silently shorter, and
  the UI only knows `catalogFailed` for a hard error.
- Screenshot flow: `scripts/screenshots/scenarios/flows/onboarding/create-space.ts`
  clicks `create-space-base` and picks `^USD`.

## Fix

1. Replace the three `Select`s with one searchable combobox
   (`Popover` + `Command` or the logic of `CurrencySelect`), match on code,
   name and symbol, with a capped, scrollable list (`max-h-64`). Put popular
   currencies (USD, EUR, GBP, the user's locale currency) first.
2. Keep `data-testid="create-space-base"` on the trigger and make the option
   text start with the code so the screenshot flow keeps working.
3. Make the incomplete catalog visible: have `/currencies/catalog` say whether
   it came from the fallback (e.g. `source: "fallback"` in the response), show
   "Showing a short list, the currency service is unavailable" with Retry,
   and let the user type any ISO 4217 code manually in that case
   (`spacefill` already accepts any code matching `currencyCodePattern`).
4. Docs: `docs/spaces.md` if it describes currency selection.

## Tests (write first)

- Combobox: typing "yen" or "JPY" finds Japanese Yen; list length is capped;
  already chosen codes are excluded.
- Go: `Source.Catalog` marks fallback results (fake HTTP server returns 500).
- Frontend: fallback flag shows the notice and the manual code entry.

## Done when

Any currency can be found by typing, and the user can tell when the list is
incomplete.
