# 10 — Require at least one account to create a space

## Problem

A space with no account cannot hold a transaction, and the first screens after
onboarding are empty. Today nothing requires an account: the create modal
lets you go on with zero, and the backend accepts it.

## Where

- Frontend rule: `resources/ts/components/features/spaces/create/draft.ts`
  `draftProblems` returns `'name' | 'baseCurrency' | 'accounts' | 'balances'`.
  `'accounts'` only fires for an incomplete account (empty name or currency),
  never for zero accounts. `CreateSpaceModal.tsx` disables "Next" on
  `accounts` / `balances` problems.
- Backend rule: `internal/spaces/spacefill/spacefill.go` `Fill.normalize`
  validates each account but not the count; `Fill.empty()` treats a request
  with no accounts as fine.
- The note says "in onboarding": the modal is used both for the first space
  and later ones (`create-space-copy` flow). Copying from another space brings
  that space's accounts.
- Screenshot flow: `scripts/screenshots/scenarios/flows/onboarding/create-space.ts`
  (shot `entities-invalid` currently expects a nameless account to block).
- API tests: search `internal/httpserver` for space creation tests that post
  no accounts (`spaces_test.go`, `token_presets_test.go`).

## Fix

1. Decided: the rule applies to **every** new space made through the modal,
   not only the first. Copying accounts from another space counts.
2. `draftProblems`: add `'noAccounts'` when the final content (own plus
   copied) has no account; show "Add at least one account, such as your
   wallet or bank card" under the accounts section; keep "Next" disabled.
   Consider pre-adding one empty account row in the draft.
3. Backend: `normalize` returns `invalid("accounts", "At least one account is
   required.")` when `len(Accounts) == 0` for a filled request. A bare
   `POST /api/spaces` with a name only (`Fill.empty()`) has no content step;
   check whether API clients and the demo seeder use it and keep it working.
4. Update `docs/spaces.md` and `docs/api.md`.

## Tests (write first)

- `draft.test.ts`: no accounts → problem; one valid account → none; copied
  accounts count.
- `spacefill` Go test: a fill with currencies but no accounts is a 422 on
  `accounts`.
- Screenshot flow: add an `entities-no-account` shot (detail) and keep the
  rest passing.

## Done when

The UI and API both refuse a space with no account, with a message that says
what to add.
