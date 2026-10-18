# 01 — "Use this address" button on the public URL field

## Problem

The first-run server setup pre-fills the public URL with the address the admin
opened the app at (`window.location.origin`). Administration → System → Server
does not: it is an empty input with a placeholder. An admin who clears it, or
whose URL was never set because `config.toml` was edited by hand, has to type
it.

## Where

- `resources/ts/pages/admin/system.tsx` `TextRow` (used only for `app-url`):
  `Input` plus a Save button that shows when the draft changed.
- `resources/ts/components/features/onboarding/AppUrlStep.tsx`: already computes
  `browserUrl = window.location.origin` and uses it as default and placeholder.
- Strings: `settings:system.server.*` in `locales/{en,ru}/settings.json`.

## Fix

1. Add an optional `suggestion?: string` prop to `TextRow`. When set and the
   draft differs from it, show a small button inside/next to the input
   ("Use {{url}}", icon `Globe` or `LocateFixed`) that sets the draft. It does
   not save: the usual Save button still appears.
2. Pass `window.location.origin` from the `app-url` row. Hide the button when
   `readOnly` (`config_writable === false`).
3. Reuse the same button in `AppUrlStep` instead of the silent default, so both
   places behave the same. Pull the button into one small component
   (`components/features/settings/UseBrowserUrlButton.tsx`) if that is
   cleaner than a prop.
4. New string in `en` and `ru` (`lib/i18n-keys.test.ts` checks both).

## Tests (write first)

- `system.test.tsx` (or a new `TextRow` test): with an empty value the button
  is shown, a click fills the input with `window.location.origin` and shows
  Save, and nothing is sent until Save.
- Read-only config: no button.
- When the draft already equals the origin: no button.

## Done when

An admin can fill the URL in one click on both screens and the field never
saves without an explicit Save.
