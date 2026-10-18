# 08 — Make the default categories and tags usable

## Problem

The presets read like abstractions: "Rent" is too exact (and next to "Home"),
"Shopping" covers everything and so means nothing, "Utilities" is unclear to
many users. Tags ("Essential", "Optional", "Recurring", "One-time",
"Business", …) are not localised at all.

## Where

- `internal/seed/reference.go`: `defaultCategories` (13 expense, 6 income;
  `#OTHER` and `#OTHER_INCOME` are the defaults of their type) and
  `defaultTags` (plain English strings).
- Category names are tokens (`#RENT`) that the SPA translates from
  `resources/ts/locales/{en,ru}/defaults.json` (`categories.RENT`, …).
  Tags have no tokens, so they are stored and shown as English in every
  language.
- Consumers: `internal/httpserver/api/catalog/catalog.go` (presets endpoint),
  `internal/spaces/spacefill/spacefill.go` `withDefaultCategories` (adds the
  `#OTHER*` defaults), the create-space modal (`create/draft.ts`,
  `draft.test.ts` uses `#RENT`), `internal/seed/seed_test.go` (asserts
  `DefaultCategories()[0].Name == "#RENT"`, `DefaultTags()[0] == "Essential"`).
- `internal/seed/demo.go` looks categories up by name (`catByName`).

## Fix

1. Agree a new list (needs a decision, see below). Direction: fewer, clearer
   buckets that match how people think about spending, e.g. Housing (rent,
   mortgage, repairs), Bills & subscriptions (instead of "Utilities":
   electricity, internet, phone, streaming), Groceries, Eating out, Transport,
   Health, Fun, Clothes & personal care (instead of "Shopping"), Gifts &
   charity, Travel, Education, Family & kids, Other. Income: Salary, Side
   income, Investments, Gifts, Refunds, Other income.
2. Add a one-line hint per category in `defaults.json` ("Electricity, water,
   internet, phone"), shown as helper text in the create-space modal and the
   category form, so a name never has to carry the whole meaning.
3. Tokenise tags the same way (`#ESSENTIAL`) and translate them in `en` and
   `ru`; pick a smaller, more useful set (e.g. Essential, Subscription,
   Shared, Business, Vacation) and drop tags that duplicate a category or a
   recurring flag ("Recurring", "One-time", "Planned").
4. Update `seed_test.go`, `draft.test.ts`, `demo.go` references, and any
   screenshot or e2e flow that picks a category by its name.
5. Because this is pre-release: change the list in place, no migration for
   existing spaces. Mention it in the release notes.
6. Update `docs/` where the default categories are listed.

## Tests (write first)

- Seed: every token has a translation in both locales (extend
  `lib/i18n-keys.test.ts` or add a test that reads `reference.go`, like
  `backend-contract.test.ts`).
- Exactly one default per type; names unique per type.
- Tags are tokens and translated; the create-space modal shows translated
  names in `ru`.

## Open question

Which final list do you want? Draft it in this file before coding.
